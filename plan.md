1. **Optimize `buildRowLine` in `pkg/tdbcli/cli/output.go`**:
   - Change `buildRowLine` to accept pre-calculated cell widths as a slice.
   - Avoid calling `runewidth.StringWidth(cell)` inside `buildRowLine`.
   - Update `renderTable` to calculate and store the cell widths when it iterates through the rows, passing them to `buildRowLine`.
   - Use a pre-allocated strings.Builder to write the spaces directly in `padCell` (or inline the padding loop inside `buildRowLine`) instead of `strings.Repeat(" ", pad)`, which creates extra string allocations.

2. **Optimize `buildBorder` in `pkg/tdbcli/cli/output.go`**:
   - Write `"─"` directly in a loop to `strings.Builder` instead of using `strings.Repeat("─", w+2)`, which creates a temporary string on every call for every column.

3. **Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.**
   - Run `go vet ./...` and `go test ./...`.
   - Update the journal `.jules/bolt.md`.
   - Compile the binary locally `go build -o mytdb ./cmd/tdb` and remove it afterwards.
   - Remove `go/telemetry` and any temp files.

4. **Submit**:
   - PR Title: `⚡ Bolt: Optimize table rendering output buffering`
   - PR Description with the required Bolt sections.
