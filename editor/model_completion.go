package editor

func (m Model) CompletionState() CompletionState {
	return cloneCompletionState(m.completionState)
}

func (m Model) SetCompletionState(state CompletionState) Model {
	m.completionState = cloneCompletionState(state)
	m.recomputeCompletionFilter(&m.completionState)
	return m
}

func (m Model) ClearCompletion() Model {
	m.completionState = CompletionState{}
	return m
}
