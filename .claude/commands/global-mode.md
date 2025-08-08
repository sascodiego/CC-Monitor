Enable global context mode for architecture analysis:

1. VALIDATE PURPOSE:
   ```
   Valid reasons for global mode:
   - Architecture audit
   - SOLID compliance check
   - Project initialization
   - System-wide refactoring
   - Security audit
   ```

2. PREPARE FOR HIGH TOKEN USAGE:
   - Run /checkpoint first
   - Clear unnecessary context
   - Prepare for multiple compacts

3. SWITCH TO GLOBAL:
   ```markdown
   MODE: GLOBAL
   PURPOSE: [specific reason]
   DOCUMENTATION TARGET: docs/[file].md
   EXPECTED DURATION: [estimate]
   TOKEN STRATEGY: Aggressive compaction
   ```

4. LOAD FULL PROJECT STRUCTURE:
   - Read project tree
   - Identify all sectors
   - Map dependencies
   - Note architectural patterns

5. IMPORTANT REMINDER:
   "Remember: Document findings to docs/, then return to sectorial for implementation"