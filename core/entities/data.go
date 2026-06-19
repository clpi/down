package entities

type (
	Data interface {
		// Id
		File
		// Name
		// Desc
		// Created
		// Parent
		// Workspace
		// Tags
		file() File
		tag(string)
		setName(string)
		setDesc(string)
		addToWorkspace(string)
		addTag(Tag)
	}
	Asset interface {
		// Id
		// Name
		// Desc
		// Created
		// Parent
		// Tags
		// Links
		// Groups

		fmt() string

		addTask()

		addGroup()

		addFile()

		addLink()

		addTag()

		setRoot()
	}
)
