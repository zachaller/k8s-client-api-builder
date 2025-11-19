# DSL Enhancements - Implementation Summary

## Overview

Successfully implemented three major DSL features that were previously listed as limitations:

1. ✅ **Array Indexing** - Access array elements by index
2. ✅ **Arithmetic Operations** - Perform calculations in expressions  
3. ✅ **String Concatenation** - Combine strings and values

## Features Implemented

### 1. Array Indexing

**Syntax:** `$(.path.to.array[index])`

**Capabilities:**
- Numeric indices: `$(.spec.items[0])`
- Variable indices: `$(.spec.items[.spec.selectedIndex])`
- Nested arrays: `$(.spec.pods[0].containers[1])`
- Bounds checking with clear error messages

**Implementation:**
- New expression type: `ExprArrayIndex`
- Parser function: `parseArrayIndexExpr()`
- Evaluator function: `evaluateArrayIndex()`
- Supports slices, arrays, and maps

### 2. Arithmetic Operations

**Syntax:** `$(.path + .other.path)`, `$((.a + .b) * .c)`

**Supported Operators:**
- `+` Addition
- `-` Subtraction
- `*` Multiplication
- `/` Division
- `%` Modulo

**Capabilities:**
- Integer and floating-point arithmetic
- Parentheses for precedence control
- Mixed type operations (auto-conversion)
- Division by zero protection
- Returns integers when possible

**Implementation:**
- Enhanced binary expression handling
- New function: `performArithmetic()`
- Type conversion helpers: `toInt()`, `toFloat64()`
- Operator precedence via parentheses

### 3. String Concatenation

**Syntax:** `$(.path + "-" + .other.path)`

**Capabilities:**
- Concatenate multiple values
- Mix paths and string literals
- Automatic type-to-string conversion
- Handles any number of elements

**Implementation:**
- New expression type: `ExprConcat`
- Parser function: `tryParseConcatExpr()`
- Evaluator function: `evaluateConcat()`
- Separate parsing path to avoid infinite recursion

## Technical Details

### Parser Enhancements

**New Expression Types:**
```go
const (
    ExprPath       // Existing
    ExprFunction   // Existing
    ExprBinary     // Enhanced
    ExprLiteral    // Existing
    ExprArrayIndex // NEW
    ExprConcat     // NEW
)
```

**Enhanced Expression Structure:**
```go
type Expression struct {
    Type     ExprType
    Path     string
    Index    *Expression      // NEW: For array indexing
    Function string
    Args     []string
    Operator string
    Left     *Expression
    Right    *Expression
    Elements []*Expression    // NEW: For concatenation
}
```

### Key Functions Added

**Parser (`pkg/dsl/parser.go`):**
- `parseArrayIndexExpr()` - Parse array index syntax
- `tryParseConcatExpr()` - Parse concatenation expressions
- `parseNonConcatExpression()` - Prevent infinite recursion
- `hasMultiplePlusOutsideParens()` - Detect concatenation vs arithmetic
- `findOperatorPosition()` - Find operators outside parentheses

**Evaluator (`pkg/dsl/evaluator.go`):**
- `evaluateArrayIndex()` - Execute array indexing
- `evaluateConcat()` - Execute string concatenation
- `performArithmetic()` - Execute arithmetic operations
- `toInt()` - Convert values to integers

### Parsing Strategy

The parser uses a multi-phase approach to handle ambiguity:

1. **Check for concatenation** (if contains `+` with quotes or multiple `+`)
2. **Check for arithmetic** (operators: `-`, `*`, `/`, `%`, then `+`)
3. **Check for functions** (but not parenthesized expressions)
4. **Check for array indexing**
5. **Check for comparisons**
6. **Fall back to paths or literals**

This ordering prevents:
- Infinite recursion in concat parsing
- Misinterpreting `+` in concatenation as arithmetic
- Treating parenthesized expressions as function calls

## Test Coverage

Comprehensive test suite added (`pkg/dsl/dsl_test.go`):

### Array Indexing Tests
- Simple numeric index
- Variable index
- Nested array access
- Out of bounds error handling

### Arithmetic Tests
- Addition, subtraction, multiplication, division, modulo
- Complex expressions with parentheses
- Mixed type operations

### String Concatenation Tests
- Simple two-part concatenation
- Multiple part concatenation
- Mixing paths and literals

### Combined Features Tests
- Array indexing with variable indices
- Concatenation with array elements

**All tests passing:** ✅

## Examples

### Array Indexing
```yaml
# Access first container
container: $(.spec.containers[0].name)

# Use variable index
selected: $(.spec.items[.spec.selectedIndex])
```

### Arithmetic
```yaml
# Calculate replicas
replicas: $(.spec.baseReplicas * .spec.scaleFactor)

# Complex calculation
cpu: $((.spec.base + .spec.increment) * 100)m
```

### String Concatenation
```yaml
# Build resource name
name: $(.metadata.namespace + "-" + .metadata.name)

# Build image tag
image: $("myregistry.io/" + .spec.app + ":" + .spec.version)
```

## Documentation Updates

1. **DSL Reference** (`docs/dsl-reference.md`)
   - Moved features from "Limitations" to "Advanced Features"
   - Added comprehensive examples for each feature
   - Updated limitations section

2. **Advanced Examples** (`examples/advanced-dsl-features.md`)
   - Real-world usage examples
   - Best practices
   - Error handling
   - Migration guide

## Breaking Changes

**None.** All changes are backward compatible:
- Existing expressions continue to work
- New syntax is additive only
- No changes to existing APIs

## Performance

- **Array Indexing**: O(1) constant time
- **Arithmetic**: Minimal overhead, native operations
- **Concatenation**: O(n) where n is number of elements
- **Parsing**: One-time cost, expressions are cached

## Known Limitations

1. **No operator precedence**: Arithmetic is evaluated left-to-right
   - **Workaround**: Use parentheses `$((.a + .b) * .c)`
   
2. **No array slicing**: Can't do `.spec.items[0:5]`
   - **Future enhancement**: Could be added with similar approach

3. **No nested function calls**: Can't do `$(lower(trim(.spec.name)))`
   - **Current**: Functions take single arguments
   - **Future enhancement**: Parse nested function calls

## Future Enhancements

Potential additions building on this foundation:

1. **Operator Precedence**: Implement proper precedence rules
2. **Array Slicing**: Support `[start:end]` syntax
3. **Nested Functions**: Allow `$(func1(func2(.path)))`
4. **More Operators**: `&&`, `||`, `!` for boolean logic
5. **Ternary Operator**: `$(condition ? true_value : false_value)`
6. **Array Methods**: `$(.spec.items.length)`, `$(.spec.items.first)`

## Migration Path

For users who were working around these limitations:

**Before:**
```yaml
# Had to pre-calculate in instance
spec:
  calculatedReplicas: 6
  fullName: "prod-my-app"
  firstContainer: "nginx"
```

**After:**
```yaml
# Calculate dynamically in template
replicas: $(.spec.baseReplicas * 2)
name: $(.spec.env + "-" + .metadata.name)
container: $(.spec.containers[0].name)
```

## Conclusion

These enhancements significantly increase the power and flexibility of the DSL, enabling:

- **Dynamic resource generation** based on calculations
- **Flexible naming** through string composition
- **Array manipulation** for multi-resource scenarios
- **Reduced boilerplate** in instance files
- **More expressive templates** with less duplication

The implementation maintains backward compatibility while opening up new possibilities for platform abstraction design.

---

**Implementation Date:** 2024
**Test Status:** ✅ All tests passing
**Documentation:** ✅ Complete
**Backward Compatibility:** ✅ Maintained

