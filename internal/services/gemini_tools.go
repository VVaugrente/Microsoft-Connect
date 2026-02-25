package services


func GetGeminiTools() []GeminiFunctionDecl {
    microsoftTools := GetMicrosoftTools()
    geminiTools := make([]GeminiFunctionDecl, 0, len(microsoftTools))

    for _, t := range microsoftTools {
        geminiTools = append(geminiTools, GeminiFunctionDecl{
            Name:        t.Name,
            Description: t.Description,
            Parameters:  t.InputSchema,
        })
    }

    return geminiTools
}