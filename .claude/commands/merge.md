# Unified Merge Command - /merge

## `.claude/commands/merge.md`

```markdown
Intelligent merge to target branch with automatic preparation:

USAGE: /merge $ARGUMENTS
Where $ARGUMENTS is the target branch (default: main)

1. PARSE TARGET:
   ```bash
   TARGET_BRANCH="${ARGUMENTS:-main}"
   CURRENT_BRANCH=$(git branch --show-current)
   
   if [[ "$CURRENT_BRANCH" == "$TARGET_BRANCH" ]]; then
     echo "ERROR: Already on target branch $TARGET_BRANCH"
     exit 1
   fi
   ```

2. PRE-MERGE VALIDATION:
   ```bash
   # Check for uncommitted changes
   if [[ -n $(git status --porcelain) ]]; then
     echo "⚠️ Uncommitted changes detected"
     git add -A
     git commit -m "WIP: Pre-merge checkpoint"
   fi
   
   # Check if project has tests configured
   if [[ -f "CLAUDE.md" ]]; then
     TEST_CMD=$(grep -oP '(?<=test_command: ).*' CLAUDE.md 2>/dev/null || echo "")
     if [[ -n "$TEST_CMD" ]]; then
       echo "🧪 Running tests: $TEST_CMD"
       eval "$TEST_CMD" || {
         echo "❌ Tests failed. Fix before merging."
         exit 1
       }
     else
       echo "ℹ️ No test command configured in CLAUDE.md"
     fi
   fi
   ```

3. ANALYZE MERGE TYPE:
   ```bash
   # Count commits to merge
   COMMIT_COUNT=$(git rev-list --count $TARGET_BRANCH..HEAD)
   
   # Check for WIP commits
   WIP_COUNT=$(git log $TARGET_BRANCH..HEAD --oneline | grep -c "WIP:" || true)
   
   # Determine merge strategy
   if [[ $WIP_COUNT -gt 0 ]]; then
     STRATEGY="rebase-interactive"
     echo "📝 Found $WIP_COUNT WIP commits to clean"
   elif [[ $COMMIT_COUNT -eq 1 ]]; then
     STRATEGY="fast-forward"
     echo "⚡ Single commit - fast forward possible"
   elif [[ $COMMIT_COUNT -le 3 ]]; then
     STRATEGY="merge-clean"
     echo "✅ Clean commits - standard merge"
   else
     STRATEGY="squash-option"
     echo "📦 Multiple commits - consider squashing"
   fi
   ```

4. CLEAN HISTORY IF NEEDED:
   ```bash
   # REQUIRES: User interaction for rebase
   if [[ "$STRATEGY" == "rebase-interactive" ]]; then
     echo "🧹 Cleaning history..."
     echo "ℹ️ MANUAL ACTION REQUIRED: Editor will open for interactive rebase"
     
     # Interactive rebase to squash WIPs
     git rebase -i $TARGET_BRANCH
     # User must manually mark commits for squashing
     
     echo "✅ History cleaned"
   fi
   
   if [[ "$STRATEGY" == "squash-option" ]]; then
     echo "Found $COMMIT_COUNT commits. Squash into one? (y/n): "
     read SQUASH
     if [[ "$SQUASH" == "y" ]]; then
       echo "ℹ️ MANUAL ACTION REQUIRED: Mark commits to squash in editor"
       git rebase -i $TARGET_BRANCH
     fi
   fi
   ```

5. SYNC WITH TARGET:
   ```bash
   echo "🔄 Syncing with $TARGET_BRANCH..."
   
   # Fetch latest
   git fetch origin $TARGET_BRANCH
   
   # Rebase onto target
   git rebase origin/$TARGET_BRANCH || {
     echo "⚠️ CONFLICTS DETECTED!"
     echo "MANUAL ACTION REQUIRED:"
     echo "1. Resolve conflicts in affected files"
     echo "2. Run: git add <resolved files>"
     echo "3. Run: git rebase --continue"
     echo "4. Run: /merge $TARGET_BRANCH again"
     exit 1
   }
   
   # Run tests again if configured
   if [[ -n "$TEST_CMD" ]]; then
     echo "🧪 Re-running tests after rebase..."
     eval "$TEST_CMD" || {
       echo "❌ Tests failed after rebase"
       exit 1
     }
   fi
   ```

6. GENERATE MERGE SUMMARY:
   ```markdown
   MERGE_MSG="Merge: $CURRENT_BRANCH → $TARGET_BRANCH
   
   ## Changes:
   $(git log $TARGET_BRANCH..HEAD --oneline | head -10)
   
   ## Statistics:
   - Commits: $COMMIT_COUNT
   - Files changed: $(git diff --stat $TARGET_BRANCH..HEAD | tail -1)
   
   ## Modified Areas:
   $(git diff --name-only $TARGET_BRANCH..HEAD | cut -d'/' -f1-2 | sort -u)
   "
   ```

7. EXECUTE MERGE:
   ```bash
   echo "🎯 Executing merge to $TARGET_BRANCH..."
   
   # Checkout target
   git checkout $TARGET_BRANCH
   
   # Pull latest
   git pull origin $TARGET_BRANCH
   
   # Perform merge based on strategy
   case $STRATEGY in
     fast-forward)
       git merge $CURRENT_BRANCH --ff-only
       ;;
     *)
       git merge $CURRENT_BRANCH --no-ff -m "$MERGE_MSG"
       ;;
   esac
   
   # Verify merge success
   if [[ $? -eq 0 ]]; then
     echo "✅ Merge successful!"
   else
     echo "❌ Merge failed"
     git merge --abort
     exit 1
   fi
   ```

8. POST-MERGE ACTIONS:
   ```bash
   # Build validation if configured
   if [[ -f "CLAUDE.md" ]]; then
     BUILD_CMD=$(grep -oP '(?<=build_command: ).*' CLAUDE.md 2>/dev/null || echo "")
     if [[ -n "$BUILD_CMD" ]]; then
       echo "🔨 Running build: $BUILD_CMD"
       eval "$BUILD_CMD" || echo "⚠️ Build failed on $TARGET_BRANCH"
     fi
   fi
   
   # Push to remote (REQUIRES: push permissions)
   echo "Push to remote? (y/n): "
   read PUSH
   if [[ "$PUSH" == "y" ]]; then
     git push origin $TARGET_BRANCH || {
       echo "❌ Push failed. Check your permissions for $TARGET_BRANCH"
       echo "You may need: git push --set-upstream origin $TARGET_BRANCH"
     }
     echo "📤 Pushed to origin/$TARGET_BRANCH"
   fi
   
   # Cleanup (REQUIRES: force push permissions if branch was rebased)
   echo "Delete branch '$CURRENT_BRANCH'? (y/n): "
   read DELETE
   if [[ "$DELETE" == "y" ]]; then
     git branch -d $CURRENT_BRANCH 2>/dev/null || {
       echo "⚠️ Cannot delete branch with unmerged changes"
       echo "Use: git branch -D $CURRENT_BRANCH (force delete)"
     }
     
     # Delete remote branch (REQUIRES: delete permissions)
     git push origin --delete $CURRENT_BRANCH 2>/dev/null || {
       echo "⚠️ Cannot delete remote branch (may need permissions)"
     }
     echo "🗑️ Branch cleaned up"
   else
     # Return to original branch
     git checkout $CURRENT_BRANCH
     echo "↩️ Returned to $CURRENT_BRANCH"
   fi
   ```

9. UPDATE CONTEXT:
   ```markdown
   Update SESSION_LOG.md:
   [HH:MM] MERGE: $CURRENT_BRANCH → $TARGET_BRANCH
   - Strategy: $STRATEGY
   - Commits merged: $COMMIT_COUNT
   - Conflicts: None/Resolved
   - Build: Pass/Fail/Not configured
   - Pushed: Yes/No
   
   Update GIT_CONTEXT.md:
   - Last merge: $CURRENT_BRANCH → $TARGET_BRANCH
   - Timestamp: $(date)
   - Clean history: Yes/No
   ```

10. FINAL SUMMARY:
    ```
    🎉 MERGE COMPLETE
    
    Source: $CURRENT_BRANCH
    Target: $TARGET_BRANCH
    Strategy: $STRATEGY
    Commits: $COMMIT_COUNT merged
    Pushed: ${PUSHED:-No}
    Branch deleted: ${DELETED:-No}
    
    Next steps:
    - Continue with new feature: /session-start [feature]
    - Review merged code: git log --oneline -5
    - Check CI/CD status if applicable
    ```

Response: "Merge to $TARGET_BRANCH complete."
```

## 🔧 Configuration in CLAUDE.md

Add these optional configurations to your CLAUDE.md file:

```markdown
## BUILD & TEST CONFIGURATION
test_command: go test ./...
build_command: cargo build --release
lint_command: clang-format -i **/*.cpp

## Alternative examples:
# Go projects:
test_command: go test -v ./...
build_command: go build -o bin/app

# Rust projects:
test_command: cargo test
build_command: cargo build --release

# C++ projects:
test_command: make test
build_command: make release

# Python projects:
test_command: pytest
build_command: python setup.py build

# No tests:
test_command: 
build_command: make
```

## ⚠️ Required Permissions

### Git Permissions Needed:
- **Local repository**: Read/write access
- **Push to protected branches**: May require admin rights
- **Delete remote branches**: May require maintainer rights
- **Force push (after rebase)**: May be restricted on main/master

### If Permission Denied:
```bash
# For protected branch push:
ERROR: protected branch hook declined
SOLUTION: Create PR or request maintainer to merge

# For force push after rebase:
ERROR: non-fast-forward updates were rejected  
SOLUTION: Use --force-with-lease or request permissions

# For branch deletion:
ERROR: insufficient permission for deleting branch
SOLUTION: Ask repo admin to delete or keep branch
```

## 🎯 Usage Examples

### Simple Merge to Main
```bash
/merge
# Automatically merges current branch to main
```

### Merge to Specific Branch
```bash
/merge develop
# Merges current branch to develop
```

### Merge to Release Branch
```bash
/merge release/v2.0
# Merges current branch to release/v2.0
```

## 🔄 Smart Behaviors

### Auto-Detection Features
1. **WIP Detection**: Automatically offers to clean WIP commits
2. **Conflict Resolution**: Guides through conflicts if they occur
3. **Test Validation**: Won't merge if tests fail
4. **History Cleaning**: Suggests squashing if too many commits
5. **Branch Cleanup**: Offers to delete merged branches

### Safety Features
1. **Pre-merge Backup**: Commits any uncommitted changes
2. **Test Gates**: Runs tests before AND after rebase
3. **Rollback Ready**: Can abort merge at any point
4. **Push Confirmation**: Asks before pushing to remote
5. **Branch Protection**: Won't delete branches with unpushed commits

## 📊 Integration with Vibe Flow

### During Development
```bash
# Work on feature
/checkpoint
/commit-smart

# Ready to merge
/merge              # Merges to main by default
```

### Multi-Branch Workflow
```bash
# Feature → Develop → Main
/merge develop      # First merge to develop
git checkout develop
/merge main         # Then merge develop to main
```

### Quick Fix Flow
```bash
# On fix branch
/commit-smart "fix: Resolve issue"
/merge              # Direct to main
# Branch deleted automatically
```

## 🚀 Advanced Options

### Force Strategies (Future Enhancement)
```bash
/merge main --squash        # Force squash all commits
/merge main --no-ff         # Force merge commit
/merge main --ff-only       # Force fast-forward only
/merge main --no-tests      # Skip test validation (dangerous!)
```

### Merge with Context
```bash
/merge main "Feature complete: OAuth implementation"
# Uses provided message as merge commit message
```

## 💡 Benefits of Unified Command

1. **Single Command**: No need to remember multiple merge commands
2. **Intelligent**: Adapts strategy based on branch state
3. **Safe**: Multiple validation gates
4. **Clean**: Handles history cleaning automatically
5. **Complete**: From validation to cleanup in one command

---

**USAGE**: Simply type `/merge` to merge current branch to main, or `/merge [branch]` to merge to a specific branch. The command handles everything else automatically.