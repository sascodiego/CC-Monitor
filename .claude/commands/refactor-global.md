System-wide refactoring with documentation:

1. ENABLE GLOBAL MODE:
   /global-mode "System refactoring: [type]"

2. ANALYZE REFACTORING SCOPE:
   Types:
   - Dependency upgrade
   - Pattern migration
   - Technology change
   - Performance optimization
   - Security hardening

3. CREATE REFACTORING PLAN:
   Save to docs/REFACTORING_PLAN.md:
   ```markdown
   # Refactoring Plan - [type]
   
   ## Scope
   - Affected sectors: [list]
   - Files to change: [count]
   - Risk level: [low/medium/high]
   
   ## Steps
   1. [Sector]: [changes]
   2. [Sector]: [changes]
   
   ## Rollback Plan
   [How to revert if needed]
   ```

4. EXECUTE BY SECTOR:
   - Return to sectorial mode
   - Apply changes sector by sector
   - Commit each sector separately

5. VALIDATE:
   - Run tests
   - Check architectural compliance
   - Document results