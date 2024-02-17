package messagetype

// MessageType is an enum that represents the possible message types
type MessageType string

const (
	ActionRequest         MessageType = "ActionRequest"
	Chat                              = "Chat"
	ConfirmationCard                  = "ConfirmationCard"
	Context                           = "Context"
	InternalSearchQuery               = "InternalSearchQuery"
	InternalSearchResult              = "InternalSearchResult"
	Disengaged                        = "Disengaged"
	InternalLoaderMessage             = "InternalLoaderMessage"
	Progress                          = "Progress"
	RenderCardRequest                 = "RenderCardRequest"
	RenderContentRequest              = "RenderContentRequest"
	AdsQuery                          = "AdsQuery"
	SemanticSerp                      = "SemanticSerp"
	GenerateContentQuery              = "GenerateContentQuery"
	SearchQuery                       = "SearchQuery"
	GeneratedCode                     = "GeneratedCode"
	// Still exists? \/
	Message       = ""
	CustomMessage = "CustomMessage"
	Image         = "Image"
)
