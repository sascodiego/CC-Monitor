Isolated fix within specific sector:

1. SCOPE VALIDATION:
   ```markdown
   TARGET SECTOR: [sector]
   ALLOWED FILES: [pattern]
   FORBIDDEN: Files outside sector
   FIX TYPE: [bug/performance/refactor]
   ```

2. ISOLATE CONTEXT:
   - Load only sector files
   - Identify dependencies
   - Mark external interfaces as read-only

3. APPLY FIX:
   - Make changes within sector
   - Validate no external impacts
   - Run sector-specific tests

4. COMMIT ATOMICALLY:
   ```bash
   git add [sector files only]
   git commit -m "fix([sector]): [description]"
   ```

5. VERIFY ISOLATION:
   Confirm no files outside sector were modified