package rcj

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/nathanhack/ecc/linearblock"
	"github.com/nathanhack/ecc/linearblock/ldpc/rcj"
	mat "github.com/nathanhack/sparsemat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Girth uint
var Count uint
var Force bool
var Threads uint
var Verbose bool

var RCJRun = func(cmd *cobra.Command, args []string) {

	if Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if Count <= 1 {
		fmt.Println("required: count >1")
		return
	}

	if Count&(Count-1) != 0 {
		fmt.Println("required: count be a power of 2")
		return
	}

	if Girth%2 == 1 {
		fmt.Println("required: girth%2==0")
		return
	}

	if Girth < 4 {
		fmt.Println("required: girth>=4")
		return
	}

	//we build the graph for that can contain the requested node counts
	g := rcj.Build(ctx, int(Girth), int(Count))
	if g == nil {
		fmt.Println("Unable to create RCJ Graph: ")
		return
	}

	H := makeHFrom(len(g.NodeMap[rcj.CheckNode]), len(g.NodeMap[rcj.VarNode]), g.Nodes)

	rcjEcc := linearblock.SystematicLinearBlock(ctx, H, int(Threads))

	logrus.Infof("RCJ(%v,%v) Message Size:%v Parity Size:%v  Codeword Size:%v  Code Rate: %v", int(Girth), int(Count), rcjEcc.MessageLength(), rcjEcc.ParitySymbols(), rcjEcc.CodewordLength(), rcjEcc.CodeRate())

	bs, err := json.Marshal(rcjEcc)
	if err != nil {
		fmt.Println("Unable to serialize the LDPC: ", err)
		return
	}

	err = os.WriteFile(args[0], bs, 0644)
	if err != nil {
		fmt.Println("unable to write file: ", err)
	}
	logrus.Info("Done")
}

func truncateGraph(ctx context.Context, girth, requestedCount, targetVarNodes int, g *rcj.Graph) {
	if int(requestedCount*girth/2) == len(g.NodeMap[rcj.CheckNode]) &&
		targetVarNodes == len(g.NodeMap[rcj.VarNode]) {
		// we have what was asked for so just return
		return
	}

	logrus.Infof("WARNING: Truncating RCJ Graph to match requested size")
	logrus.Infof("Currently %v checkNodes target:%v removing %v extra checkNodes", len(g.NodeMap[rcj.CheckNode]), int(requestedCount*girth/2), len(g.NodeMap[rcj.CheckNode])-int(requestedCount*girth/2))
	// now with the graph in hand we truncate to the size we need
	// to trim we first get rid of the extra checkNodes
	for int(requestedCount*girth/2) < len(g.NodeMap[rcj.CheckNode]) {

		select {
		case <-ctx.Done():
			return
		default:
		}
		g.RemoveNode(len(g.NodeMap[rcj.CheckNode])-1, rcj.CheckNode)
	}

	logrus.Info("Clean up Zero connected VarNodes")
	// now we clean up any varNodes have zero connection
	for i := 0; i < len(g.Nodes); {
		n := g.Nodes[i]
		if n.Type != rcj.VarNode || len(n.Connections) != 0 {
			i++
			continue
		}

		g.RemoveNode(n.Index, rcj.VarNode)
	}

	targetDiff := targetVarNodes - len(g.NodeMap[rcj.VarNode])
	switch {
	case targetDiff == 0:
		return
	case targetDiff < 0:
		logrus.Info("Too many VarNodes, removing extras")
		// we still need to get rid of VarNodes but
		// we'll want to remove non loop VarNodes only
		for _, n := range g.NodeMap[rcj.VarNode] {
			if n.Loop > 0 {
				continue
			}
			if !g.RemoveNode(n.Index, rcj.VarNode) {
				panic("this shouldn't happen")
			}
			if targetVarNodes == len(g.NodeMap[rcj.VarNode]) {
				break
			}
		}
	case targetDiff > 0:
		//we need to add in some VarNodes, unfortunately
		// these will have only one connections not ideal

		logrus.Info("More VarNodes needed")
		sort.Slice(g.Nodes, func(i, j int) bool {
			ni := g.Nodes[i]
			nic := ni.Type == rcj.CheckNode
			nj := g.Nodes[j]
			njc := nj.Type == rcj.CheckNode

			if nic && njc {
				return ni.Index < nj.Index
			}

			return nic
		})

		for i := 0; i < girth/2 && 0 < targetDiff; i++ {
			for j := i; j < len(g.NodeMap[rcj.CheckNode]) && 0 < targetDiff; j += girth / 2 {
				n := g.CreateNode(rcj.VarNode)
				g.Nodes[j].ConnectTo(n)
				targetDiff--
			}
		}
	}

	if targetVarNodes != len(g.NodeMap[rcj.VarNode]) {
		logrus.Fatalf("unable to remove enough VarNodes to meet request %v, %v was best effort", targetVarNodes, len(g.NodeMap[rcj.VarNode]))
	}

	removeIndicesGaps(g)

	for i := 0; i < len(g.NodeMap[rcj.CheckNode]); i++ {
		_, has := g.NodeMap[rcj.CheckNode][i]
		if !has {
			panic("here")
		}
	}

	for i := 0; i < len(g.NodeMap[rcj.VarNode]); i++ {
		_, has := g.NodeMap[rcj.VarNode][i]
		if !has {
			panic("here")
		}
	}
}

func removeIndicesGaps(g *rcj.Graph) {

	logrus.Info("Removing Indices Gap")

	maxIndex := len(g.NodeMap[rcj.CheckNode]) - 1
	over := make(map[int]*rcj.Node)

	for i, n := range g.NodeMap[rcj.CheckNode] {
		if i <= maxIndex {
			continue
		}

		over[i] = n
	}

	currentIndex := 0
	for _, n := range over {
		for currentIndex <= maxIndex {
			if _, has := g.NodeMap[rcj.CheckNode][currentIndex]; has {
				currentIndex++
				continue
			}
			//we'll move n from i to currentIndex
			tmp := append([]int{}, n.Connections...)

			for _, c := range tmp {
				n.DisconnectFrom(g.NodeMap[rcj.VarNode][c])
			}

			delete(g.NodeMap[rcj.CheckNode], n.Index)
			n.Index = currentIndex
			g.NodeMap[rcj.CheckNode][n.Index] = n
			currentIndex++

			for _, c := range tmp {
				n.ConnectTo(g.NodeMap[rcj.VarNode][c])
			}

			break
		}
	}

	// next we first fix the varNodes
	maxIndex = len(g.NodeMap[rcj.VarNode]) - 1
	over = make(map[int]*rcj.Node)

	for i, n := range g.NodeMap[rcj.VarNode] {
		if i <= maxIndex {
			continue
		}

		over[i] = n
	}

	currentIndex = 0
	for _, n := range over {
		for currentIndex <= maxIndex {
			if _, has := g.NodeMap[rcj.VarNode][currentIndex]; has {
				currentIndex++
				continue
			}
			//we'll move n from i to currentIndex
			tmp := append([]int{}, n.Connections...)

			for _, c := range tmp {
				n.DisconnectFrom(g.NodeMap[rcj.CheckNode][c])
			}

			delete(g.NodeMap[rcj.VarNode], n.Index)
			n.Index = currentIndex
			g.NodeMap[rcj.VarNode][n.Index] = n
			currentIndex++

			for _, c := range tmp {
				n.ConnectTo(g.NodeMap[rcj.CheckNode][c])
			}

			break
		}
	}
}

func makeHFrom(checkNodes, variableNodes int, allNodes []*rcj.Node) mat.SparseMat {
	//now we construct our H matrix from allNodes
	// fmt.Printf("checkNodes:%v, variableNodes:%v\n", checkNodes, variableNodes)
	H := mat.DOKMat(checkNodes, variableNodes)
	for _, node := range allNodes {
		if node.Type == rcj.CheckNode {
			for _, n := range node.Connections {
				H.Set(node.Index, n, 1)
			}
		}
	}
	return H
}


/*
wnhack@fedora:~/projects/ecc/ecc$ go run main.go c l l rcj test_rcj.json -g 32 -c 1024
INFO[0000] Building RCJ(32,1024)                        
WARN[0004] Only 7936 rows of 8192 linearly independent (diff:-256) 
INFO[0009] RCJ(32,1024) Message Size:4864 Parity Size:7936  Codeword Size:12800  Code Rate: 0.38 
INFO[0009] Done                                         
wnhack@fedora:~/projects/ecc/ecc$ go run main.go c l l rcj rcj_32_65536.json -g 32 -c 65536 -v
INFO[0000] Building RCJ(32,65536)                       
DEBU[0002] Creating generator matrix from H matrix      
DEBU[0002] Preparing matrix for Gaussian-Jordan Elimination 
DEBU[0002] Row echelon                                  
Processing Row 524267 / 524288 [-------------------------------------------------------------------------------------------------------------------------------------->] 100.00% 151 p/s ETA 0sDEBU[13881] Consolidating 16384 linear dependent rows to bottom of matrix 
Processing Row 524288 / 524288 Done                                                                                                                                                            
WARN[13881] Only 507904 rows of 524288 linearly independent (diff:-16384) 
DEBU[13881] Reduced row echelon                          
Processing Row 524288 / 524288 Done                                                                                                                                                            
DEBU[17301] Gaussian-Jordan Elimination complete         
DEBU[23976] Validating Row Reduced Matrix                
DEBU[30481] Extracting A Matrix from Row Reduced Matrix  
DEBU[33592] Creating Generator Matrix                    
DEBU[42556] Generator Matrix complete                    
INFO[42556] RCJ(32,65536) Message Size:507904 Parity Size:507904  Codeword Size:1015808  Code Rate: 0.5 
INFO[42557] Done 
*/