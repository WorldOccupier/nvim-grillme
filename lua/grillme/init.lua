local M = {}

local state = {
  file = vim.fs.joinpath(vim.fn.getcwd(), ".grillme", "session.jsonl"),
  question = nil,
  buffer = nil,
  timer = nil,
}

local function events()
  local lines = vim.fn.readfile(state.file)
  local result = {}
  for _, line in ipairs(lines) do
    local ok, value = pcall(vim.json.decode, line)
    if ok then
      table.insert(result, value)
    end
  end
  return result
end

local function pending_question()
  if vim.fn.filereadable(state.file) == 0 then
    return nil
  end
  local questions, answered = {}, {}
  for _, value in ipairs(events()) do
    if value.type == "question" then
      table.insert(questions, value)
    elseif value.type == "answer" then
      answered[value.question_id] = true
    end
  end
  for index = #questions, 1, -1 do
    local question = questions[index]
    if not answered[question.id] then
      return question
    end
  end
end

local function render(question)
  if not state.buffer or not vim.api.nvim_buf_is_valid(state.buffer) then
    return
  end
  state.question = question
  local lines = question and {
    "GrillMe",
    "",
    question.text,
    "",
    "Answer:",
    "",
  } or { "GrillMe", "", "Waiting for a question…" }
  vim.api.nvim_buf_set_lines(state.buffer, 0, -1, false, lines)
  vim.bo[state.buffer].modified = false
  if question then
    local win = vim.fn.bufwinid(state.buffer)
    if win ~= -1 then
      vim.api.nvim_win_set_cursor(win, { #lines, 0 })
      vim.api.nvim_set_current_win(win)
      vim.cmd.startinsert()
    end
  end
end

local function refresh()
  local question = pending_question()
  if (question and question.id) ~= (state.question and state.question.id) then
    render(question)
  end
end

function M.open()
  if state.buffer and vim.api.nvim_buf_is_valid(state.buffer) then
    local win = vim.fn.bufwinid(state.buffer)
    if win ~= -1 then
      vim.api.nvim_set_current_win(win)
      return
    end
  end

  vim.cmd("enew")
  state.buffer = vim.api.nvim_get_current_buf()
  vim.bo[state.buffer].buftype = "nofile"
  vim.bo[state.buffer].bufhidden = "hide"
  vim.bo[state.buffer].swapfile = false
  vim.bo[state.buffer].filetype = "markdown"
  vim.api.nvim_buf_set_name(state.buffer, "GrillMe")
  vim.keymap.set({ "n", "i" }, "<C-s>", M.submit, { buffer = state.buffer })
  render(nil)
  refresh()

  state.timer = vim.uv.new_timer()
  state.timer:start(0, 150, vim.schedule_wrap(refresh))
end

function M.submit()
  if not state.question then
    return
  end
  local lines = vim.api.nvim_buf_get_lines(state.buffer, 0, -1, false)
  local marker
  for index, line in ipairs(lines) do
    if line == "Answer:" then
      marker = index
      break
    end
  end
  local answer = marker and vim.trim(table.concat(vim.list_slice(lines, marker + 1), "\n")) or ""
  if answer == "" then
    vim.notify("Answer cannot be empty", vim.log.levels.WARN)
    return
  end
  local value = vim.json.encode({ type = "answer", question_id = state.question.id, text = answer })
  vim.fn.mkdir(vim.fs.dirname(state.file), "p")
  vim.fn.writefile({ value }, state.file, "a")
  if vim.env.HERDR_PANE_ID then
    vim.system({ vim.env.HERDR_BIN_PATH or "herdr", "pane", "close", vim.env.HERDR_PANE_ID }, { detach = true })
  end
  vim.cmd("qa")
end

return M
