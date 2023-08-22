package main

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		for _, file := range plugin.Files {
			err := generate(plugin, file)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func generate(plugin *protogen.Plugin, in *protogen.File) error {
	fileName := "GrpcModule.ts"
	out := plugin.NewGeneratedFile(fileName, in.GoImportPath)

	out.P(`import {NativeModules} from 'react-native';

const {GrpcModule} = NativeModules;

type GrpcModule = {
    setHost: (host: string) => void;
    setPort: (port: number) => void;`)

	for _, service := range in.Services {
		for _, method := range service.Methods {
			err := generateMethod(method, out)
			if err != nil {
				return err
			}
		}
	}

	out.P("}")

	for _, message := range in.Messages {
		err := generateMessage(message, out)
		if err != nil {
			return err
		}
	}

	out.P()
	out.P("export default GrpcModule as GrpcModule;")

	return nil
}

func generateMessage(message *protogen.Message, out *protogen.GeneratedFile) error {
	variables := make(map[string]string)
	lists := make(map[string][]string)

	variables["messageName"] = upper(string(message.Desc.Name()))

	// TODO: handle nested messages

	for _, field := range message.Fields {
		lists["fieldTypes"] = append(lists["fieldTypes"], lower(field.Desc.Kind().String())) // TODO: typescript type
		lists["fieldNames"] = append(lists["fieldNames"], lower(string(field.Desc.Name())))
	}

	const template = `export type $messageName$ = {
	$*fieldNames*$: $*fieldTypes*$;
};`

	out.P(format(template, &variables, &lists))

	return nil
}

func generateMethod(method *protogen.Method, out *protogen.GeneratedFile) error {
	variables := make(map[string]string)
	lists := make(map[string][]string)

	variables["methodName"] = lower(string(method.Desc.Name()))
	variables["inputKindLower"] = lower(string(method.Desc.Input().Name()))
	variables["inputKindUpper"] = upper(string(method.Desc.Input().Name()))
	variables["outputKindUpper"] = upper(string(method.Desc.Output().Name()))

	const template = "\t$methodName$: ($inputKindLower$: $inputKindUpper$) => Promise<$outputKindUpper$>;"

	out.P(format(template, &variables, &lists))

	return nil
}

func format(template string, variables *map[string]string, lists *map[string][]string) string {
	result := strings.Clone(template)
	for key, value := range *variables {
		result = strings.ReplaceAll(result, "$"+key+"$", value)
	}
	for key := range *lists {
		start := 0
		for start != -1 {
			oldStart := start
			start = strings.Index(result[oldStart:], "$*"+key+"*$")
			if start == -1 {
				break
			}
			start += oldStart
			lineStart := strings.LastIndex(result[:start], "\n")
			if lineStart == -1 {
				break
			}
			lineStart++
			lineEnd := strings.Index(result[start:], "\n")
			if lineEnd == -1 {
				break
			}
			lineEnd += start
			line := result[lineStart:lineEnd]

			var variablesInLine []string
			var listLength int
			for i := 0; i < len(line); i++ {
				if line[i] == '$' && line[i+1] == '*' && i+2 < len(line) {
					variableStart := i + 2
					variableEnd := strings.Index(line[variableStart:], "*$")
					if variableEnd == -1 {
						continue
					}
					variableEnd += variableStart
					variableName := line[variableStart:variableEnd]
					variable, ok := (*lists)[variableName]
					if !ok {
						continue
					}
					if contains(variablesInLine, variableName) {
						continue
					}
					if variableLen := len(variable); listLength == 0 || listLength > variableLen {
						listLength = variableLen
					}
					variablesInLine = append(variablesInLine, variableName)
				}
			}

			var formattedLine string
			for i := 0; i < listLength; i++ {
				currentLine := strings.Clone(line)
				for j := 0; j < len(variablesInLine); j++ {
					currentLine = strings.ReplaceAll(currentLine, "$*"+variablesInLine[j]+"*$", (*lists)[variablesInLine[j]][i])
				}
				if i != 0 {
					formattedLine += "\n"
				}
				formattedLine += currentLine
			}
			result = strings.ReplaceAll(result, line, formattedLine)
		}
	}
	return result
}

func lower(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToLower(str[:1]) + str[1:]
}

func upper(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
