local root = vim.fn.getcwd()
local directory = vim.fn.tempname()
vim.fn.mkdir(vim.fs.joinpath(directory, ".grillme"), "p")
vim.opt.runtimepath:append(root)

local file = vim.fs.joinpath(directory, ".grillme", "session.jsonl")
vim.fn.writefile({
  vim.json.encode({ type = "question", id = "q1", text = "Done?" }),
  "not json",
  vim.json.encode({ type = "question", id = "q2", text = "Pending?" }),
  vim.json.encode({ type = "answer", question_id = "q1", text = "Yes" }),
}, file)

local session = require("grillme.session")
local events = session.read_events(file)
assert(#events == 3)
local pending = session.pending_questions(file)
assert(#pending == 1 and pending[1].id == "q2")

session.append_answers(file, pending, { "Later" })
local appended = vim.json.decode(vim.fn.readfile(file)[5])
assert(appended.type == "answer")
assert(appended.question_id == "q2")
assert(appended.text == "Later")
assert(appended.timestamp:match("^%d%d%d%d%-%d%d%-%d%dT%d%d:%d%d:%d%dZ$"))
assert(#session.pending_questions(file) == 0)
vim.cmd("qa!")
