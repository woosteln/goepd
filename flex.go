package epd

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"

	"github.com/jinzhu/copier"
	"github.com/kjk/flex"
)

var FlexDirectionMap = map[string]flex.FlexDirection{
	flex.FlexDirectionToString(flex.FlexDirectionColumn):        flex.FlexDirectionColumn,
	flex.FlexDirectionToString(flex.FlexDirectionColumnReverse): flex.FlexDirectionColumnReverse,
	flex.FlexDirectionToString(flex.FlexDirectionRow):           flex.FlexDirectionRow,
	flex.FlexDirectionToString(flex.FlexDirectionRowReverse):    flex.FlexDirectionRowReverse,
}

var FlexWrapMap = map[string]flex.Wrap{
	flex.WrapToString(flex.WrapNoWrap):      flex.WrapNoWrap,
	flex.WrapToString(flex.WrapWrap):        flex.WrapWrap,
	flex.WrapToString(flex.WrapWrapReverse): flex.WrapWrapReverse,
}

var FlexJustifyMap = map[string]flex.Justify{
	flex.JustifyToString(flex.JustifyFlexStart):    flex.JustifyFlexStart,
	flex.JustifyToString(flex.JustifyFlexEnd):      flex.JustifyFlexEnd,
	flex.JustifyToString(flex.JustifyCenter):       flex.JustifyCenter,
	flex.JustifyToString(flex.JustifySpaceBetween): flex.JustifySpaceBetween,
	flex.JustifyToString(flex.JustifySpaceAround):  flex.JustifySpaceAround,
}

var FlexAlignMap = map[string]flex.Align{
	flex.AlignToString(flex.AlignAuto): flex.AlignAuto,
	// AlignFlexStart is "flex-start"
	flex.AlignToString(flex.AlignFlexStart): flex.AlignFlexStart,
	// AlignCenter if "center"
	flex.AlignToString(flex.AlignCenter): flex.AlignCenter,
	// AlignFlexEnd is "flex-end"
	flex.AlignToString(flex.AlignFlexEnd): flex.AlignFlexEnd,
	// AlignStretch is "strech"
	flex.AlignToString(flex.AlignStretch): flex.AlignStretch,
	// AlignBaseline is "baseline"
	flex.AlignToString(flex.AlignBaseline): flex.AlignBaseline,
	// AlignSpaceBetween is "space-between"
	flex.AlignToString(flex.AlignSpaceBetween): flex.AlignSpaceBetween,
	// AlignSpaceAround is "space-around"
	flex.AlignToString(flex.AlignSpaceAround): flex.AlignSpaceAround,
}

func FlexAlignFromString(align string) flex.Align {
	if a, ok := FlexAlignMap[align]; ok {
		return a
	}
	return flex.AlignAuto
}

func FlexDirectionFromString(direction string) flex.FlexDirection {
	if d, ok := FlexDirectionMap[direction]; ok {
		return d
	}
	return flex.FlexDirectionColumn
}

func FlexWrapFromString(wrap string) flex.Wrap {
	if w, ok := FlexWrapMap[wrap]; ok {
		return w
	}
	return flex.WrapNoWrap
}

func FlexJustifyFromString(justify string) flex.Justify {
	if j, ok := FlexJustifyMap[justify]; ok {
		return j
	}
	return flex.JustifyFlexStart
}

type NodeList []*Node

type Node struct {
	ID             string      `json:"id"`
	Type           string      `json:"type"`
	Children       NodeList    `json:"children"`
	Content        interface{} `json:"content"`
	FontSize       float64     `json:"fontSize"`
	FontWeight     string      `json:"fontWeight"`
	Width          int         `json:"width"`
	Height         int         `json:"height"`
	Padding        float32     `json:"padding"`
	Margin         float32     `json:"margin"`
	FlexDirection  string      `json:"flexDirection"`
	FlexGrow       float32     `json:"flexGrow"`
	FlexShrink     float32     `json:"flexShrink"`
	FlexWrap       string      `json:"flexWrap"`
	JustifyContent string      `json:"justifyContent"`
	AlignItems     string      `json:"alignItems"`
	AlignContent   string      `json:"alignContent"`
	AlignSelf      string      `json:"alignSelf"`
	FlexBasis      string      `json:"flexBasis"`
}

func DefaultNode() Node {
	return Node{
		FlexDirection:  flex.FlexDirectionToString(flex.FlexDirectionRow),
		FlexGrow:       0,
		FlexShrink:     1,
		JustifyContent: flex.JustifyToString(flex.JustifyFlexStart),
		AlignItems:     flex.AlignToString(flex.AlignStretch),
		AlignContent:   flex.AlignToString(flex.AlignStretch),
		AlignSelf:      flex.AlignToString(flex.AlignAuto),
		Children:       nil,
	}
}

func (n *Node) UnmarshalJSON(data []byte) error {
	type Alias Node
	tmp := Alias(DefaultNode())
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	copier.Copy(n, &tmp)
	return nil
}

var ErrNodeNotFound = errors.New("Node not found")

var PatternInt = regexp.MustCompile(`^\d+$`)

// FindNodeById provides depth first search for a child node
// with the provided id.
func (n *Node) FindNodeById(id string) (node *Node, err error) {
	for _, candidate := range n.Children {
		if candidate.ID == id {
			return candidate, nil
		}
		if child, errr := candidate.FindNodeById(id); errr == nil {
			return child, nil
		}
	}
	err = ErrNodeNotFound
	return
}

func (n *Node) Inflate(config ...*flex.Config) (flexNode *flex.Node) {

	var engine flexRenderEngine

	if len(config) > 0 {
		conf := config[0]
		flexNode = flex.NewNodeWithConfig(conf)
		engine = conf.Context.(flexRenderEngine)
	} else {
		flexNode = flex.NewNode()
	}

	flexNode.Context = n
	if n.Content != nil {
		flexNode.SetMeasureFunc(engine.Measure)
		if _, ok := n.Content.(string); ok {
			flexNode.NodeType = flex.NodeTypeText
		}
	} else {

		idx := 0
		for _, child := range n.Children {
			flexChild := child.Inflate(config...)
			flexNode.InsertChild(flexChild, idx)
			idx++
		}

	}

	hasContent := n.Content != nil

	flexNode.StyleSetFlexDirection(FlexDirectionFromString(n.FlexDirection))
	flexNode.StyleSetFlexGrow(n.FlexGrow)
	flexNode.StyleSetFlexShrink(n.FlexShrink)
	flexNode.StyleSetFlexWrap(FlexWrapFromString(n.FlexWrap))
	flexNode.StyleSetAlignContent(FlexAlignFromString(n.AlignContent))
	flexNode.StyleSetAlignItems(FlexAlignFromString(n.AlignItems))
	flexNode.StyleSetAlignSelf(FlexAlignFromString(n.AlignSelf))
	flexNode.StyleSetJustifyContent(FlexJustifyFromString(n.JustifyContent))
	if hasContent {
		flexNode.StyleSetPadding(flex.EdgeAll, n.Padding)
	}
	if PatternInt.Match([]byte(n.FlexBasis)) {
		i, _ := strconv.Atoi(n.FlexBasis)
		flexNode.StyleSetFlexBasis(float32(i))
	} else { // TODO add percent switch
		flex.NodeStyleSetFlexBasisAuto(flexNode)
	}
	flexNode.StyleSetAlignSelf(FlexAlignFromString(n.AlignSelf))

	return flexNode

}
