package completion

// const (
// 	CompletionsDefault Completions = append{Completions{}, Completions{}}
// 	CompletionsLink    Completions = append{Completions{}, Completions{}}
// 	CompletionsTask    Completions = append{Completions{}, Completions{}}
// 	CompletionsRef     Completions = append{Completions{}, Completions{}}
// 	CompletionsRef     Completions = append{Completions{}, Completions{}}
// 	CompletionsSymbol  Completions = append{Completions{}, Completions{}}
// 	CompletionsHeader  Completions = append{Completions{}, Completions{}}
// 	/// "@fdsf @erkfjf"
// 	CompletionsReference Completions = append{Completions{}, Completions{}}
// 	CompletionsTag                   = Completions{}
// 	CompletionsDefault               = iota
// 	CompletionsNone                  = iota
// )

// const (
// 	CompletionsLink Completions = iota
// 	// CompletionsEmoji
// )

// var (
// // Trigger map[string]Completions{
// //   "#": CompletionsLink,

// // }
// )

// var (
// 	Triggers = map[string]map[string]Completions{
// 		")", ".": {
// 			" ": CompletionsListOrd,
// 		},
// 		"-", "+", "*": {
// 			'a', 'z': CompletionsDefault,
// 			" ": {
// 				'a', 'z' | ' ': CompletionsList,
// 				"[": CompletionsTask,
// 			},
// 		},
// 		"`": {
// 			"`": {
// 				"`": CompletionsCodeBlock,
// 			},
// 			'a', 'z': CompletionsCode,
// 		},
// 		"[": CompletionsLink,
// 		"(": CompletionsParen,
// 		"{": CompletionsCurly,
// 		">": CompletionsQuote,
// 		"<": CompletionsHtml,
// 		"@": {
// 			" ": CompletionsAt,
// 			"a", "z": CompletionsReference,
// 		},
// 		"#": {
// 			" ": CompletionsHeader,
// 			"a", "z": CompletionsTag,
// 		},
// 	}
// )
