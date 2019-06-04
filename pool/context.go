package pool

type poolContext struct {
	compactDirty bool
}

func (context *poolContext) setCompactDirty(dirty bool) {
	context.compactDirty = dirty
}
