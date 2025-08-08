# Universal Compilation Error Analyzer - /analyze-errors

## **YOU MUST BE USE** error-amalyst agent
## **YOU CAN NOT USE** pio run
## Errors is in .last_build.log

## `.claude/commands/analyze-errors.md`

```markdown
Analyze, group, and create action plan for compilation errors from any language/toolchain:

USAGE: /analyze-errors [optional: error log file]
If no file specified, reads from last build output or clipboard

1. CAPTURE ERROR OUTPUT:
   ```bash
   # Try to get errors from various sources
   if [[ -n "$ARGUMENTS" ]]; then
     ERROR_LOG="$ARGUMENTS"
   elif [[ -f ".last_build.log" ]]; then
     ERROR_LOG=".last_build.log"
   else
     echo "Paste compilation errors (Ctrl+D when done):"
     ERROR_LOG=$(cat)
   fi
   ```

2. DETECT BUILD SYSTEM & LANGUAGE:
   ```markdown
   ANALYZING BUILD OUTPUT...
   
   Detected:
   - Build System: [PlatformIO/Make/Cargo/Go/CMake/Gradle/Maven/etc]
   - Language: [C++/Rust/Go/Java/Python/TypeScript/etc]
   - Platform: [ESP8266/Linux/Windows/embedded/etc]
   - Compiler: [gcc/clang/rustc/go/javac/tsc/etc]
   ```

3. PARSE & CATEGORIZE ERRORS:
   ```markdown
   # Error Categories Identified
   
   ## CRITICAL ERRORS (Build Stoppers)
   - Count: [X]
   - Type: [Syntax/Type/Declaration/Linking]
   - Impact: Build completely blocked
   
   ## WARNINGS (Non-blocking)
   - Count: [Y]
   - Type: [Unused variables/Deprecation/etc]
   - Impact: Build continues but issues present
   
   ## DEPENDENCY ISSUES
   - Missing libraries/packages
   - Version conflicts
   - Import/include problems
   ```

4. GROUP ERRORS BY PATTERN:
   ```markdown
   # Error Analysis Report
   
   ## Group 1: Type/Declaration Errors
   ### Root Cause: Missing type definition 'WiFiConnectionState'
   
   **Affected Files:**
   - MultiWiFiManager.h:242 (declaration)
   - MultiWiFiManager.cpp:455 (implementation)
   
   **Error Chain:**
   1. Type 'WiFiConnectionState' not declared
   2. → Template instantiation fails for Result<WiFiConnectionState, String>
   3. → Method signature mismatch between .h and .cpp
   
   **Likely Cause:** Missing include or undefined type
   
   ---
   
   ## Group 2: Compiler Feature Errors
   ### Root Cause: Exception handling disabled
   
   **Affected Files:**
   - MultiWiFiManager.cpp:97
   
   **Error Details:**
   - Trying to use try/catch with -fno-exceptions
   - Platform constraint (embedded system)
   
   ---
   
   ## Group 3: Operator/Comparison Errors
   ### Root Cause: Type mismatch in comparison
   
   **Affected Files:**
   - MultiWiFiManager.cpp:315
   
   **Error Details:**
   - Comparing const char* with String object
   - Need proper string comparison method
   
   ---
   
   ## Group 4: Undefined Variable Errors
   ### Root Cause: Variable 'configData' not in scope
   
   **Affected Files:**
   - MultiWiFiManager.cpp:592
   
   **Error Details:**
   - Variable used but never declared
   - Possibly missing from function parameters or class members
   ```

5. CREATE DEPENDENCY GRAPH:
   ```markdown
   # Error Dependencies
   
   WiFiConnectionState undefined
     ├→ Result<WiFiConnectionState, String> fails
     ├→ getConnectionStatus() declaration invalid
     └→ Method implementation mismatch
   
   Exception handling disabled
     └→ All try/catch blocks fail
   
   String comparison issue
     └→ Logic errors in network selection
   ```

6. GENERATE ACTION PLAN:
   ```markdown
   # 🎯 ACTION PLAN
   
   ## Priority 1: Fix Type Definitions [BLOCKING]
   Estimated Time: 15 minutes
   Sector: /services/network
   
   ### Steps:
   1. /sector-switch /services/network
   2. Search for WiFiConnectionState definition:
      - Check if it should be an enum/struct/typedef
      - Look in WiFi library headers
      - May need to define it ourselves
   3. Add missing definition to appropriate header
   4. /checkpoint
   5. Rebuild to verify this group of errors is resolved
   
   Command Sequence:
   ```bash
   /sector-fix /services/network
   # Fix WiFiConnectionState definition
   /commit-smart "fix(network): Add missing WiFiConnectionState type"
   ```
   
   ---
   
   ## Priority 2: Handle Exception Constraints [PLATFORM]
   Estimated Time: 20 minutes
   Sector: /services/network
   
   ### Steps:
   1. Replace try/catch with error codes
   2. Use Result<T,E> pattern instead
   3. Add compile flag check for embedded
   
   Command Sequence:
   ```bash
   /checkpoint
   # Replace exception handling
   /commit-smart "fix(network): Replace exceptions with Result pattern"
   ```
   
   ---
   
   ## Priority 3: Fix String Comparisons [LOGIC]
   Estimated Time: 10 minutes
   Sector: /services/network
   
   ### Steps:
   1. Change comparison to use String::equals() or strcmp()
   2. Verify all string comparisons in file
   
   Command Sequence:
   ```bash
   # Fix string comparisons
   /commit-smart "fix(network): Correct string comparison operators"
   ```
   
   ---
   
   ## Priority 4: Resolve Undefined Variables [SCOPE]
   Estimated Time: 10 minutes
   Sector: /services/network
   
   ### Steps:
   1. Trace where configData should come from
   2. Add proper variable declaration/parameter
   3. Check for similar issues in same function
   
   Command Sequence:
   ```bash
   # Fix undefined variables
   /commit-smart "fix(network): Add missing configData variable"
   /clean-history  # Clean up all fixes
   /merge  # Merge to main when all errors resolved
   ```
   ```

7. CREATE FIX VERIFICATION SCRIPT:
   ```bash
   # Save as .claude/verify-fixes.sh
   #!/bin/bash
   
   echo "🔍 Verifying fixes..."
   
   # Run build command from CLAUDE.md
   BUILD_CMD=$(grep -oP '(?<=build_command: ).*' CLAUDE.md)
   
   # Capture new errors
   $BUILD_CMD 2>&1 | tee .last_build_new.log
   
   # Compare error counts
   OLD_ERRORS=$(grep -c "error:" .last_build.log)
   NEW_ERRORS=$(grep -c "error:" .last_build_new.log)
   
   echo "Errors before: $OLD_ERRORS"
   echo "Errors after: $NEW_ERRORS"
   echo "Fixed: $((OLD_ERRORS - NEW_ERRORS))"
   
   if [[ $NEW_ERRORS -eq 0 ]]; then
     echo "✅ All compilation errors resolved!"
   else
     echo "⚠️ Still have $NEW_ERRORS errors remaining"
   fi
   ```

8. GENERATE SUMMARY REPORT:
   ```markdown
   # Compilation Error Analysis Summary
   
   ## Statistics
   - Total Errors: 6
   - Total Warnings: 2
   - Affected Files: 3
   - Estimated Fix Time: 55 minutes
   
   ## Root Causes (Prioritized)
   1. Missing type definitions (causes 3 errors)
   2. Platform constraints not handled (1 error)
   3. Incorrect operators used (1 error)  
   4. Undefined variables (1 error)
   
   ## Recommended Approach
   1. Start with type definitions (will fix multiple errors)
   2. Use /sector-fix to isolate changes
   3. Test after each fix group
   4. Use /checkpoint frequently
   5. /merge when all green
   
   ## Pattern Recognition
   ⚠️ This appears to be an embedded system project (ESP8266)
   Common issues for embedded:
   - No exception handling
   - Limited standard library
   - Memory constraints
   - Special string types
   
   Consider creating embedded-specific coding guidelines in CLAUDE.md
   ```

9. SAVE ANALYSIS:
   ```bash
   # Save full analysis
   cat > docs/BUILD_ERROR_ANALYSIS_$(date +%Y%m%d_%H%M).md << EOF
   [Full analysis report]
   EOF
   
   # Update context
   echo "[$(date)] Analyzed $TOTAL_ERRORS errors" >> SESSION_LOG.md
   ```

10. SUGGEST NEXT COMMAND:
    ```
    📋 ANALYSIS COMPLETE
    
    Found: 6 errors, 2 warnings
    Root causes: 4 distinct issues
    Estimated fix time: ~1 hour
    
    Suggested next command:
    > /sector-fix /services/network
    
    Or to see the full plan:
    > cat docs/BUILD_ERROR_ANALYSIS_[timestamp].md
    ```

Response: "Error analysis complete. Action plan created. Ready to start fixes with /sector-fix"
```

## 🎯 Features of the Universal Analyzer

### Language Agnostic Detection
- Automatically detects language from error patterns
- Works with any compiler output format
- Adapts analysis based on detected toolchain

### Intelligent Grouping
```markdown
Groups by:
1. Root cause (same underlying issue)
2. File/module (locality of changes)
3. Error type (syntax/semantic/linking)
4. Dependency chain (cascading errors)
```

### Smart Action Plan Generation
- Prioritizes blockers first
- Groups related fixes together
- Estimates time per fix group
- Suggests command sequences

## 📊 Example Usage Patterns

### C++ Project (like your ESP8266)
```bash
pio run 2>&1 | tee .last_build.log
/analyze-errors .last_build.log
# Generates ESP8266-specific fixes
```

### Rust Project
```bash
cargo build 2>&1 | tee .last_build.log
/analyze-errors
# Detects Rust patterns, suggests cargo-specific fixes
```

### Go Project
```bash
go build ./... 2>&1 | tee .last_build.log
/analyze-errors
# Recognizes Go import issues, suggests go mod fixes
```

### TypeScript Project
```bash
tsc --noEmit 2>&1 | tee .last_build.log
/analyze-errors
# Identifies type errors, suggests interface fixes
```

## 🔄 Integration with Vibe Flow

```bash
# When build fails
/analyze-errors
> Generates action plan

# Follow the plan
/sector-fix /services/network
> Fix Priority 1 issues
/checkpoint

# Test fix
make 2>&1 | tee .last_build.log
/analyze-errors
> Shows remaining errors

# Continue until clean
/commit-smart "fix: Resolve all compilation errors"
/merge
```

## 💡 Advanced Features

### Pattern Learning
The analyzer remembers common patterns:
```markdown
# Added to DECISION_CACHE.md
## Common Error Patterns
- ESP8266: No exceptions, use Result<T,E>
- String comparisons: Use .equals() not ==
- Missing types: Check library headers first
```

### Preventive Suggestions
```markdown
After analysis, suggests additions to CLAUDE.md:
## Coding Guidelines
- NEVER use try/catch in embedded code
- ALWAYS use String::equals() for comparisons
- DEFINE all custom types in Types.h
```

---

**USAGE**: Run `/analyze-errors` after any build failure. It will analyze, group, and create a systematic fix plan using all the vibe coding commands.