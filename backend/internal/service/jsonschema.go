package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

// validateJSON validates data against a JSON Schema subset, deliberately
// dependency-free. Supported keywords: type, required, properties, items,
// enum, minLength, maxLength, minimum, maximum. Unknown keywords are ignored
// so manifests can adopt newer schema features without breaking older skm
// builds. Paths in error messages use JSON pointer style (e.g. /claimId).
func validateJSON(schemaJSON json.RawMessage, data []byte) error {
	var schema any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return fmt.Errorf("invalid input schema in manifest: %w", err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("input is not valid JSON: %w", err)
	}
	return validateSchemaValue(schema, value, "")
}

func validateSchemaValue(schema, value any, path string) error {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil // non-object schema (true/nil) imposes no constraints
	}

	if expected, hasType := schemaMap["type"]; hasType {
		if err := checkType(expected, value, path); err != nil {
			return err
		}
	}

	if enumValues, hasEnum := schemaMap["enum"]; hasEnum {
		allowed, ok := enumValues.([]any)
		if !ok {
			return nil
		}
		for _, candidate := range allowed {
			if jsonEqual(candidate, value) {
				return validateNested(schemaMap, value, path)
			}
		}
		return fmt.Errorf("%s: value must be one of %s", pathOrRoot(path), jsonString(enumValues))
	}

	return validateNested(schemaMap, value, path)
}

func validateNested(schemaMap map[string]any, value any, path string) error {
	switch typed := value.(type) {
	case string:
		if minLength, ok := numberValue(schemaMap["minLength"]); ok && float64(len([]rune(typed))) < minLength {
			return fmt.Errorf("%s: string is shorter than minLength %s", pathOrRoot(path), jsonString(minLength))
		}
		if maxLength, ok := numberValue(schemaMap["maxLength"]); ok && float64(len([]rune(typed))) > maxLength {
			return fmt.Errorf("%s: string is longer than maxLength %s", pathOrRoot(path), jsonString(maxLength))
		}
	case float64:
		if minimum, ok := numberValue(schemaMap["minimum"]); ok && typed < minimum {
			return fmt.Errorf("%s: value is less than minimum %s", pathOrRoot(path), jsonString(minimum))
		}
		if maximum, ok := numberValue(schemaMap["maximum"]); ok && typed > maximum {
			return fmt.Errorf("%s: value is greater than maximum %s", pathOrRoot(path), jsonString(maximum))
		}
	case map[string]any:
		if required, ok := schemaMap["required"].([]any); ok {
			for _, item := range required {
				key, isString := item.(string)
				if !isString {
					continue
				}
				if _, present := typed[key]; !present {
					return fmt.Errorf("%s: missing required property %q", pathOrRoot(path), key)
				}
			}
		}
		if properties, ok := schemaMap["properties"].(map[string]any); ok {
			for key, propertySchema := range properties {
				propertyValue, present := typed[key]
				if !present {
					continue
				}
				if err := validateSchemaValue(propertySchema, propertyValue, path+"/"+key); err != nil {
					return err
				}
			}
		}
	case []any:
		if itemSchema, hasItems := schemaMap["items"]; hasItems {
			for index, item := range typed {
				if err := validateSchemaValue(itemSchema, item, fmt.Sprintf("%s/%d", path, index)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkType(expected any, value any, path string) error {
	names := make([]string, 0, 1)
	switch typed := expected.(type) {
	case string:
		names = append(names, typed)
	case []any:
		for _, item := range typed {
			if name, ok := item.(string); ok {
				names = append(names, name)
			}
		}
	default:
		return nil
	}
	for _, name := range names {
		if typeMatches(name, value) {
			return nil
		}
	}
	return fmt.Errorf("%s: expected type %s, got %s", pathOrRoot(path), strings.Join(names, "|"), jsonTypeName(value))
}

func typeMatches(name string, value any) bool {
	switch name {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == float64(int64(number))
	case "number":
		_, ok := value.(float64)
		return ok
	case "null":
		return value == nil
	default:
		return true
	}
}

func jsonTypeName(value any) string {
	switch value.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func jsonEqual(a, b any) bool {
	left, errLeft := json.Marshal(a)
	right, errRight := json.Marshal(b)
	return errLeft == nil && errRight == nil && string(left) == string(right)
}

func numberValue(value any) (float64, bool) {
	number, ok := value.(float64)
	return number, ok
}

func pathOrRoot(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

func jsonString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}
