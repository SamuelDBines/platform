package nunchucks

type memoryLoader struct {
	files map[string]string
}

// MemoryLoader creates a loader backed by an in-memory map.
// Keys are template paths (for example: "base.njk").
func MemoryLoader(files map[string]string) Loader {
	cpy := make(map[string]string, len(files))
	for k, v := range files {
		cpy[k] = v
	}
	return &memoryLoader{files: cpy}
}

func (m *memoryLoader) TypeName() string { return "memory" }

func (m *memoryLoader) Source(name string) LoaderResponse {
	if _, ok := m.files[name]; !ok {
		return LoaderResponse{Err: "No file found: " + name, Res: name}
	}
	return LoaderResponse{Err: "", Res: name}
}

func (m *memoryLoader) Read(name string) LoaderResponse {
	v, ok := m.files[name]
	if !ok {
		return LoaderResponse{Err: "No file found: " + name, Res: name}
	}
	return LoaderResponse{Err: "", Res: v}
}
