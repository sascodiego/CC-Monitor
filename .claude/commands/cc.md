# /commit-context

Commit context files without violating sector boundaries.

## Usage
```
/commit-context [message]
```

Special commit that bypasses sector restrictions for context files only.

## Implementation

### 1. VALIDATE SECTOR MODE
```bash
# Check if we're in sector mode
if [[ -f ".claude/context/SECTOR_PLAN.md" ]]; then
  ACTIVE_SECTOR=$(grep "ACTIVE_SECTOR:" .claude/context/SECTOR_PLAN.md | cut -d' ' -f2)
  echo "📐 Active sector: $ACTIVE_SECTOR"
  echo "📝 Creating context-only commit (sector rules suspended for .claude/)"
fi
```

### 2. STAGE ONLY CONTEXT FILES
```bash
# Stage ONLY .claude/context files (nothing from sectors)
git add .claude/context/*.md
git add .claude/CLAUDE.md
git add .claude/commands/*.md

# Verify no sector files are staged
STAGED_FILES=$(git diff --cached --name-only)
SECTOR_VIOLATIONS=$(echo "$STAGED_FILES" | grep -v "^\.claude/" || true)

if [[ -n "$SECTOR_VIOLATIONS" ]]; then
  echo "❌ ERROR: Non-context files staged:"
  echo "$SECTOR_VIOLATIONS"
  echo "Unstaging sector files..."
  git reset $SECTOR_VIOLATIONS
fi
```

### 3. CREATE CONTEXT COMMIT
```bash
# Generate context-aware commit message
MESSAGE="${ARGUMENTS:-Context checkpoint}"

if [[ -n "$ACTIVE_SECTOR" ]]; then
  COMMIT_MSG="context($ACTIVE_SECTOR): $MESSAGE"
else
  COMMIT_MSG="context: $MESSAGE"
fi

# Commit only context files
git commit -m "$COMMIT_MSG" || {
  echo "ℹ️ No context changes to commit"
  exit 0
}
```

### 4. LOG CONTEXT COMMIT
```markdown
Update SESSION_LOG.md:
[HH:MM] CONTEXT_COMMIT: $COMMIT_MSG
- Preserved feature context
- Active sector unchanged: $ACTIVE_SECTOR
- Context files: $(git diff HEAD^ --name-only | wc -l)
```

## Response
"Context committed. Sector boundaries maintained. Continue working in $ACTIVE_SECTOR"

## Purpose
- Allows committing context files without violating sector boundaries
- Preserves feature context while maintaining architectural discipline
- Enables better documentation of decisions and progress

## When to Use
- After making important architecture decisions
- When documenting feature context
- Before switching sectors
- To preserve analysis results