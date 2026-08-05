package model

// Capabilities defines exact functional features supported by a registered LLM or embedding model.
type Capabilities struct {
	Streaming        bool `json:"streaming"`
	ToolCalling      bool `json:"tool_calling"`
	StructuredOutput bool `json:"structured_output"`
	Vision           bool `json:"vision"`
	Reasoning        bool `json:"reasoning"`
	Embedding        bool `json:"embedding"`
	Reranking        bool `json:"reranking"`
	MaxContextWindow int  `json:"max_context_window"`
}

// ModelRequirement specifies the minimum capabilities needed for a task.
type ModelRequirement struct {
	NeedToolCalling      bool `json:"need_tool_calling"`
	NeedStructuredOutput bool `json:"need_structured_output"`
	NeedVision           bool `json:"need_vision"`
	NeedReasoning        bool `json:"need_reasoning"`
	NeedEmbedding        bool `json:"need_embedding"`
	NeedReranking        bool `json:"need_reranking"`
	MinContextWindow     int  `json:"min_context_window"`
}

// Satisfies checks whether the model capabilities fulfill the requirement.
func (c Capabilities) Satisfies(req ModelRequirement) bool {
	if req.NeedToolCalling && !c.ToolCalling {
		return false
	}
	if req.NeedStructuredOutput && !c.StructuredOutput {
		return false
	}
	if req.NeedVision && !c.Vision {
		return false
	}
	if req.NeedReasoning && !c.Reasoning {
		return false
	}
	if req.NeedEmbedding && !c.Embedding {
		return false
	}
	if req.NeedReranking && !c.Reranking {
		return false
	}
	if req.MinContextWindow > 0 && c.MaxContextWindow < req.MinContextWindow {
		return false
	}
	return true
}
