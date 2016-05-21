// Code generated by "stringer -type=ResponseKind"; DO NOT EDIT

package gen

import "fmt"

const _ResponseKind_name = "RawResponseModelResponseMapResponseRawMapResponseArrayResponseRawArrayResponseEmptyResponse"

var _ResponseKind_index = [...]uint8{0, 11, 24, 35, 49, 62, 78, 91}

func (i ResponseKind) String() string {
	if i < 0 || i >= ResponseKind(len(_ResponseKind_index)-1) {
		return fmt.Sprintf("ResponseKind(%d)", i)
	}
	return _ResponseKind_name[_ResponseKind_index[i]:_ResponseKind_index[i+1]]
}

var _ResponseKindNameToValue_map = map[string]ResponseKind{
	_ResponseKind_name[0:11]:  0,
	_ResponseKind_name[11:24]: 1,
	_ResponseKind_name[24:35]: 2,
	_ResponseKind_name[35:49]: 3,
	_ResponseKind_name[49:62]: 4,
	_ResponseKind_name[62:78]: 5,
	_ResponseKind_name[78:91]: 6,
}

func ResponseKindString(s string) (ResponseKind, error) {
	if val, ok := _ResponseKindNameToValue_map[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to ResponseKind values", s)
}
