package main

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const template = `import {NativeModules} from 'react-native';

const {GrpcModule} = NativeModules;

type GrpcModule = {
    setHost: (host: string) => void;
    setPort: (port: number) => void;
	$*methods*$
};

$*messages*$

export default GrpcModule as GrpcModule;`

var kinds = map[protoreflect.Kind]string{
	protoreflect.BoolKind:     "boolean",
	protoreflect.Int32Kind:    "number",
	protoreflect.Sint32Kind:   "number",
	protoreflect.Uint32Kind:   "number",
	protoreflect.Int64Kind:    "number",
	protoreflect.Sint64Kind:   "number",
	protoreflect.Uint64Kind:   "number",
	protoreflect.Sfixed32Kind: "number",
	protoreflect.Fixed32Kind:  "number",
	protoreflect.FloatKind:    "number",
	protoreflect.Sfixed64Kind: "number",
	protoreflect.Fixed64Kind:  "number",
	protoreflect.DoubleKind:   "number",
	protoreflect.StringKind:   "string",
	// TODO: handle these
	// protogen.EnumKind: ""
	// protogen.BytesKind: "",
	// protogen.MessageKind: "",
	// protogen.GroupKind:   "",
}

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
	fileName := "grpcModule.ts"
	out := plugin.NewGeneratedFile(fileName, in.GoImportPath)

	lists := make(map[string][]string)

	generateServices(in.Services, &lists)

	generateMessages(in.Messages, &lists)

	out.P(formatLists(template, &lists))

	return nil
}

func generateServices(services []*protogen.Service, lists *map[string][]string) {
	for _, service := range services {
		for _, method := range service.Methods {
			(*lists)["methods"] = append((*lists)["methods"], generateMethod(method))
		}
	}
}

func generateMessages(messages []*protogen.Message, lists *map[string][]string) {
	for _, message := range messages {
		(*lists)["messages"] = append((*lists)["messages"], generateMessage(message))
	}
}

func generateMessage(message *protogen.Message) string {
	variables := make(map[string]string)
	lists := make(map[string][]string)

	variables["messageName"] = upper(string(message.Desc.Name()))

	var nestedMessages string
	for _, nestedMessage := range message.Messages {
		nestedMessages += generateMessage(nestedMessage)
	}
	variables["nestedMessages"] = nestedMessages

	for _, field := range message.Fields {
		fieldKind := field.Desc.Kind()
		fieldType, ok := kinds[fieldKind]
		if !ok {
			if fieldKind == protoreflect.MessageKind {
				lists["fieldTypes"] = append(lists["fieldTypes"], upper(string(field.Message.Desc.Name())))
				lists["fieldNames"] = append(lists["fieldNames"], lower(string(field.Desc.Name())))
			}
			continue
		}
		lists["fieldTypes"] = append(lists["fieldTypes"], fieldType)
		lists["fieldNames"] = append(lists["fieldNames"], lower(string(field.Desc.Name())))
	}

	var template = `export type $messageName$ = {
	$*fieldNames*$: $*fieldTypes*$;
};`
	if len(nestedMessages) > 0 {
		template += `
		$nestedMessages$`
	}

	return format(template, &variables, &lists)
}

func generateMethod(method *protogen.Method) string {
	variables := make(map[string]string)

	variables["methodName"] = lower(string(method.Desc.Name()))
	variables["inputKindLower"] = lower(string(method.Desc.Input().Name()))
	variables["inputKindUpper"] = upper(string(method.Desc.Input().Name()))
	variables["outputKindUpper"] = upper(string(method.Desc.Output().Name()))

	const template = "\t$methodName$: ($inputKindLower$: $inputKindUpper$) => Promise<$outputKindUpper$>;"

	return formatVariables(template, &variables)
}

func formatVariables(template string, variables *map[string]string) string {
	result := template
	for key, value := range *variables {
		result = strings.ReplaceAll(result, "$"+key+"$", value)
	}
	return result
}

func formatLists(template string, lists *map[string][]string) string {
	lines := strings.Split(template, "\n")
	resultLines := strings.Builder{}

	for i, line := range lines {
		var start int
		lineAmount := -1
		lineVariables := make(map[string][]string)
		for start != -1 {
			oldStart := start
			start = strings.Index(line[start:], "$*")
			if start == -1 {
				break
			}
			start += oldStart

			end := strings.Index(line[start:], "*$")
			if end == -1 {
				break
			}
			end += start

			name := line[start+2 : end]
			if variable, ok := (*lists)[name]; ok {
				if variableLen := len(variable); lineAmount == -1 || lineAmount > variableLen {
					lineAmount = variableLen
				}
				lineVariables[name] = variable
			}

			start = end + 2
		}

		if lineAmount == -1 {
			lineAmount = 1
		}
		for j := 0; j < lineAmount; j++ {
			currentLine := line
			for name, value := range lineVariables {
				currentLine = strings.ReplaceAll(currentLine, "$*"+name+"*$", value[j])
			}
			if i > 0 {
				resultLines.WriteByte('\n')
			}
			resultLines.WriteString(currentLine)
		}
	}

	return resultLines.String()
}

func format(template string, variables *map[string]string, lists *map[string][]string) string {
	result := formatVariables(template, variables)
	result = formatLists(result, lists)
	return result
}

func upper(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

func lower(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToLower(str[:1]) + str[1:]
}
