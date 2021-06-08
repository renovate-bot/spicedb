package parser

import (
	"container/list"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/authzed/spicedb/pkg/schemadsl/dslshape"
	"github.com/authzed/spicedb/pkg/schemadsl/input"
	"github.com/stretchr/testify/assert"
)

type testNode struct {
	nodeType   dslshape.NodeType
	properties map[string]interface{}
	children   map[string]*list.List
}

type parserTest struct {
	name     string
	filename string
}

func (pt *parserTest) input() string {
	b, err := ioutil.ReadFile(fmt.Sprintf("tests/%s.zed", pt.filename))
	if err != nil {
		panic(err)
	}

	return string(b)
}

func (pt *parserTest) tree() string {
	b, err := ioutil.ReadFile(fmt.Sprintf("tests/%s.zed.expected", pt.filename))
	if err != nil {
		panic(err)
	}

	return string(b)
}

func (pt *parserTest) writeTree(value string) {
	err := ioutil.WriteFile(fmt.Sprintf("tests/%s.zed.expected", pt.filename), []byte(value), 0644)
	if err != nil {
		panic(err)
	}
}

func createAstNode(source input.InputSource, kind dslshape.NodeType) AstNode {
	return &testNode{
		nodeType:   kind,
		properties: make(map[string]interface{}),
		children:   make(map[string]*list.List),
	}
}

func (tn *testNode) GetType() dslshape.NodeType {
	return tn.nodeType
}

func (tn *testNode) Connect(predicate string, other AstNode) AstNode {
	if tn.children[predicate] == nil {
		tn.children[predicate] = list.New()
	}

	tn.children[predicate].PushBack(other)
	return tn
}

func (tn *testNode) Decorate(property string, value string) AstNode {
	if _, ok := tn.properties[property]; ok {
		panic(fmt.Sprintf("Existing key for property %s\n\tNode: %v", property, tn.properties))
	}

	tn.properties[property] = value
	return tn
}

func (tn *testNode) DecorateWithInt(property string, value int) AstNode {
	if _, ok := tn.properties[property]; ok {
		panic(fmt.Sprintf("Existing key for property %s\n\tNode: %v", property, tn.properties))
	}

	tn.properties[property] = value
	return tn
}

func TestParser(t *testing.T) {
	var parserTests = []parserTest{
		parserTest{"empty file test", "empty"},
		parserTest{"basic definition test", "basic"},
		parserTest{"arrow test", "arrow"},
		parserTest{"multiple definition test", "multidef"},
		parserTest{"broken test", "broken"},
		parserTest{"relation missing type test", "relation_missing_type"},
		parserTest{"permission missing expression test", "permission_missing_expression"},
		parserTest{"relation invalid type test", "relation_invalid_type"},
		parserTest{"permission invalid expression test", "permission_invalid_expression"},
		parserTest{"cross tenant test", "crosstenant"},
	}

	for _, test := range parserTests {
		t.Run(test.name, func(t *testing.T) {
			root := Parse(createAstNode, input.InputSource(test.name), test.input())
			parseTree := getParseTree((root).(*testNode), 0)
			assert := assert.New(t)

			found := strings.TrimSpace(parseTree)

			if os.Getenv("REGEN") == "true" {
				test.writeTree(found)
			} else {
				expected := strings.TrimSpace(test.tree())
				if !assert.Equal(expected, found, test.name) {
					t.Log(parseTree)
				}
			}
		})
	}
}

func getParseTree(currentNode *testNode, indentation int) string {
	parseTree := ""
	parseTree = parseTree + strings.Repeat(" ", indentation)
	parseTree = parseTree + fmt.Sprintf("%v", currentNode.nodeType)
	parseTree = parseTree + "\n"

	keys := make([]string, 0)

	for key := range currentNode.properties {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		parseTree = parseTree + strings.Repeat(" ", indentation+2)
		parseTree = parseTree + fmt.Sprintf("%s = %v", key, currentNode.properties[key])
		parseTree = parseTree + "\n"
	}

	keys = make([]string, 0)

	for key := range currentNode.children {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		value := currentNode.children[key]
		parseTree = parseTree + fmt.Sprintf("%s%v =>", strings.Repeat(" ", indentation+2), key)
		parseTree = parseTree + "\n"

		for e := value.Front(); e != nil; e = e.Next() {
			parseTree = parseTree + getParseTree(e.Value.(*testNode), indentation+4)
		}
	}

	return parseTree
}
