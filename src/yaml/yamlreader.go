package yaml

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

type TreeItem struct {
	Key   interface{}
	Value interface{}
}

type TreeBranch []TreeItem

type TreeBranches []TreeBranch

type Tree struct {
	Branches TreeBranches
}

func LoadEncryptedFile(inputPath string) (*Tree, error) {
	fileBytes, err := ioutil.ReadFile(inputPath)
	if err != nil {

	}
	tree, err := LoadEncryptedYamlFile(fileBytes)
	return &tree, err
}

func LoadEncryptedYamlFile(in []byte) (Tree, error) {
	var data yaml.Node
	if err := yaml.Unmarshal(in, &data); err != nil {
	}
	var branches TreeBranches
	d := yaml.NewDecoder(bytes.NewReader(in))

	for true {
		var data yaml.Node
		err := d.Decode(&data)
		if err == io.EOF {
			break
		}
		if err != nil {

		}
		branch, err := yamlDocumentNodeToTreeBranch(data)
		if err != nil {

		}
		for i, elt := range branch {
			if elt.Key == "sops" {
				branch = append(branch[:i], branch[i+1:]...)
			}
		}
		branches = append(branches, branch)
	}
	return Tree{
		Branches: branches,
	}, nil
}

func yamlDocumentNodeToTreeBranch(in yaml.Node) (TreeBranch, error) {
	branch := make(TreeBranch, 0)
	return appendYamlNodeToTreeBranch(&in, branch)
}

func appendYamlNodeToTreeBranch(node *yaml.Node, branch TreeBranch) (TreeBranch, error) {
	var err error
	switch node.Kind {
	case yaml.DocumentNode:
		for _, item := range node.Content {
			branch, err = appendYamlNodeToTreeBranch(item, branch)
			if err != nil {

			}
		}
	case yaml.SequenceNode:
		return nil, fmt.Errorf("yaml documents not supported")
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			var keyValue interface{}
			key.Decode(&keyValue)
			valueTV, err := nodeToTreeValue(value)
			if err != nil {

			}
			branch = append(branch, TreeItem{
				Key:   keyValue,
				Value: valueTV,
			})
		}
	case yaml.ScalarNode:
		if node.ShortTag() == "!!null" {
			return branch, nil
		}
		return nil, fmt.Errorf("YAML documents that are value are not supported")
	case yaml.AliasNode:
		branch, err = appendYamlNodeToTreeBranch(node.Alias, branch)
	}
	return branch, nil
}

func nodeToTreeValue(node *yaml.Node) (interface{}, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		panic("documents should never be passed here")
	case yaml.SequenceNode:
		var result []interface{}
		for _, item := range node.Content {
			val, err := nodeToTreeValue(item)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
		return result, nil
	case yaml.MappingNode:
		branch := make(TreeBranch, 0)
		return appendYamlNodeToTreeBranch(node, branch)
	case yaml.ScalarNode:
		var result interface{}
		node.Decode(&result)
		return result, nil
	case yaml.AliasNode:
		return nodeToTreeValue(node.Alias)
	}
	return nil, nil
}

func DecryptTree(tree *Tree) (string, error) {
	hash := sha512.New()
	walk := func(branch TreeBranch) error {
		_, err := walkBranch(branch, make([]string, 0), func(in interface{}, path []string) (interface{}, error) {
			var v interface{}
			var err error
			v = "decryptedvalue"
			bytes, err := ToBytes(v)
			if err != nil {
				return nil, fmt.Errorf("Couldn't covert to bytes")
			}
			hash.Write(bytes)
			return v, nil
		})
		return err
	}
	for _, branch := range tree.Branches {
		err := walk(branch)
		if err != nil {
			return "", fmt.Errorf("Error walkig tree %s", err)
		}
	}
	return fmt.Sprintf("%X", hash.Sum(nil)), nil
}

func ToBytes(in interface{}) ([]byte, error) {
	switch in := in.(type) {
	case string:
		return ([]byte(in)), nil
	case int:
		return ([]byte(strconv.Itoa(in))), nil
	case float64:
		return []byte(strconv.FormatFloat(in, 'f', -1, 64)), nil
	case bool:
		return ([]byte(strings.Title(strconv.FormatBool(in)))), nil
	case []byte:
		return in, nil
	default:
		return nil, fmt.Errorf("Could not convert unknown type %T to butes", in)
	}
}

func walkBranch(in TreeBranch, path []string, onLeaves func(in interface{}, path []string) (interface{}, error)) (TreeBranch, error) {
	for i, item := range in {
		key, ok := item.Key.(string)
		if !ok {
			return nil, fmt.Errorf("tree centains a non-string key (tyype %T): %s. Only string key are "+
				"supported", item.Key, item.Key)
		}
		newV, err := walkValue(item.Value, append(path, key), onLeaves)
		if err != nil {
			return nil, err
		}
		in[i].Value = newV
	}
	return in, nil
}

func walkValue(in interface{}, path []string, onLeaves func(in interface{}, path []string) (interface{}, error)) (interface{}, error) {
	switch in := in.(type) {
	case string:
		return onLeaves(in, path)
	case int:
		return onLeaves(in, path)
	case float64:
		return onLeaves(in, path)
	case bool:
		return onLeaves(in, path)
	case []byte:
		return onLeaves(string(in), path)
	case TreeBranch:
		return walkBranch(in, path, onLeaves)
	case []interface{}:
		return walkSlice(in, path, onLeaves)
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("Could walk value, unknown type: %T", in)
	}
}

func walkSlice(in []interface{}, path []string, onLeaves func(in interface{}, path []string) (interface{}, error)) (interface{}, error) {
	for i, v := range in {
		newV, err := walkValue(v, path, onLeaves)
		if err != nil {
			return nil, err
		}
		in[i] = newV
	}
	return in, nil
}

func EmitPlainFile(tree *Tree) ([]byte, error) {
	var b bytes.Buffer
	e := yaml.NewEncoder(io.Writer(&b))
	e.SetIndent(4)
	for _, branch := range tree.Branches {
		var doc = yaml.Node{}
		doc.Kind = yaml.DocumentNode
		var mapping = yaml.Node{}
		mapping.Kind = yaml.MappingNode
		appendTreeBranch(branch, &mapping)
		doc.Content = append(doc.Content, &mapping)
		err := e.Encode(&doc)
		if err != nil {
			return nil, fmt.Errorf("Error marshaling to yaml %s", err)
		}
	}
	e.Close()
	return b.Bytes(), nil
}

func appendTreeBranch(branch TreeBranch, mapping *yaml.Node) {
	for _, item := range branch {
		var keyNode = &yaml.Node{}
		keyNode.Encode(item.Key)
		valueNode := treeValueToNode(item.Value)
		mapping.Content = append(mapping.Content, keyNode, valueNode)
	}
}

func treeValueToNode(in interface{}) *yaml.Node {
	switch in := in.(type) {
	case TreeBranch:
		var mapping = &yaml.Node{}
		mapping.Kind = yaml.MappingNode
		appendTreeBranch(in, mapping)
		return mapping
	case []interface{}:
		var sequence = &yaml.Node{}
		sequence.Kind = yaml.SequenceNode
		appendSequence(in, sequence)
		return sequence
	default:
		var valueNode = &yaml.Node{}
		valueNode.Encode(in)
		return valueNode
	}
}

func appendSequence(in []interface{}, sequence *yaml.Node) {
	for _, item := range in {
		itemNode := treeValueToNode(item)
		sequence.Content = append(sequence.Content, itemNode)
	}
}
