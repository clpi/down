package data

type (
	Node struct {
		ID       string
		Label    string
		Value    interface{}
		Metadata map[string]string
	}
	Tree struct {
		Node
		Children []*Tree
	}
)
