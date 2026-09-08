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

local isolated_file = vim.fs.joinpath(directory, ".grillme", "isolated.jsonl")
vim.fn.writefile({
  vim.json.encode({ type = "question", id = "a1", session_id = "session-a", text = "A?" }),
  vim.json.encode({ type = "question", id = "b1", session_id = "session-b", text = "B?" }),
  vim.json.encode({ type = "question", id = "legacy", text = "Legacy?" }),
  vim.json.encode({ type = "answer", question_id = "a1", session_id = "session-b", text = "Wrong session" }),
}, isolated_file)
local session_a = session.pending_questions(isolated_file, "session-a")
local session_b = session.pending_questions(isolated_file, "session-b")
assert(#session_a == 1 and session_a[1].id == "a1")
assert(#session_b == 1 and session_b[1].id == "b1")
assert(#session.pending_questions(isolated_file) == 3)
session.append_answers(isolated_file, session_a, { "A answer" })
local isolated_events = vim.fn.readfile(isolated_file)
local isolated_answer = vim.json.decode(isolated_events[#isolated_events])
assert(isolated_answer.session_id == "session-a")
assert(#session.pending_questions(isolated_file, "session-a") == 0)
assert(#session.pending_questions(isolated_file, "session-b") == 1)

vim.cmd("qa!")
