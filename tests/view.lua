local root = vim.fn.getcwd()
vim.opt.runtimepath:append(root)

local view = require("grillme.view")
local buffer = vim.api.nvim_create_buf(false, true)
local first_answer_line = view.render(buffer, {
  { id = "q1", text = "First line\nMore detail", recommended_answer = "Suggested\nanswer" },
  { id = "q2", text = "Second" },
})
local lines = vim.api.nvim_buf_get_lines(buffer, 0, -1, false)
assert(first_answer_line == 10)
assert(vim.deep_equal(lines, {
  "# GrillMe", "", "_Write each answer below its heading, then press `<C-s>` to submit._", "",
  "## First line", "More detail", "", "**Your answer:**", "```text", "Suggested", "answer", "```",
  "", "---", "", "## Second", "", "**Your answer:**", "```text", "", "```",
}))
local marks = vim.api.nvim_buf_get_extmarks(buffer, -1, 0, -1, { details = true })
assert(#marks == 2)
assert(marks[1][4].hl_group == "Comment")

lines[10] = "  Edited  "
lines[11] = "answer"
lines[20] = "Second answer"
local answers = view.extract_answers(lines, 2)
assert(answers[1] == "Edited  \nanswer")
assert(answers[2] == "Second answer")
assert(view.extract_answers({}, 1)[1] == "")
vim.cmd("qa!")
