local root = vim.fn.getcwd()
local directory = vim.fn.tempname()
vim.fn.mkdir(directory, "p")
vim.cmd.cd(directory)
vim.fn.mkdir(".grillme")
vim.fn.writefile({
  vim.json.encode({ type = "question", id = "q1", text = "First?", recommended_answer = "Use the default." }),
  vim.json.encode({ type = "question", id = "q2", text = "Second?" }),
}, ".grillme/session.jsonl")
vim.opt.runtimepath:append(root)

local grillme = require("grillme")
grillme.open()
assert(vim.deep_equal(vim.api.nvim_buf_get_lines(0, 0, -1, false), {
  "# GrillMe", "", "_Write each answer below its heading, then press `<C-s>` to submit._", "",
  "## First?", "", "**Your answer:**", "```text", "Use the default.", "```", "", "---", "",
  "## Second?", "", "**Your answer:**", "```text", "", "```",
}))
local marks = vim.api.nvim_buf_get_extmarks(0, -1, 0, -1, { details = true })
assert(#marks == 1)
assert(marks[1][4].hl_group == "Comment")
vim.api.nvim_win_set_cursor(0, { 9, 0 })
vim.cmd("normal! dd")
assert(vim.api.nvim_buf_get_lines(0, 8, 9, false)[1] == "```")
vim.api.nvim_buf_set_lines(0, 8, 8, false, { "One." })
vim.api.nvim_buf_set_lines(0, 17, 18, false, { "Two." })

local command = vim.cmd
vim.cmd = function() end
vim.env.HERDR_PANE_ID = nil
grillme.submit()
vim.cmd = command
local events = vim.fn.readfile(".grillme/session.jsonl")
assert(vim.json.decode(events[3]).text == "One.")
assert(vim.json.decode(events[3]).timestamp:match("^%d%d%d%d%-%d%d%-%d%dT%d%d:%d%d:%d%dZ$"))
assert(vim.json.decode(events[4]).text == "Two.")
vim.cmd("qa!")
