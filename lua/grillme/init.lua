local M = {}

local state = {
  file = vim.fs.joinpath(vim.fn.getcwd(), ".grillme", "session.jsonl"),
  questions = {},
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

local function pending_questions()
  if vim.fn.filereadable(state.file) == 0 then
    return {}
  end
  local questions, answered = {}, {}
  for _, value in ipairs(events()) do
    if value.type == "question" then
      table.insert(questions, value)
    elseif value.type == "answer" then
      answered[value.question_id] = true
    end
  end
  local pending = {}
  for _, question in ipairs(questions) do
    if not answered[question.id] then
      table.insert(pending, question)
    end
  end
  return pending
end

local function render(questions)
  if not state.buffer or not vim.api.nvim_buf_is_valid(state.buffer) then
    return
  end
  state.questions = questions
  local lines = { "GrillMe", "" }
  if #questions == 0 then
    table.insert(lines, "Waiting for a question…")
  end
  for index, question in ipairs(questions) do
    vim.list_extend(lines, { ("Question %d:"):format(index), question.text, "", ("Answer %d:"):format(index), "" })
  end
  vim.api.nvim_buf_set_lines(state.buffer, 0, -1, false, lines)
  vim.bo[state.buffer].modified = false
  if #questions > 0 then
    local win = vim.fn.bufwinid(state.buffer)
    if win ~= -1 then
      vim.api.nvim_win_set_cursor(win, { #lines, 0 })
      vim.api.nvim_set_current_win(win)
      vim.cmd.startinsert()
    end
  end
end

local function refresh()
  local questions = pending_questions()
  local ids = vim.tbl_map(function(question) return question.id end, questions)
  local previous_ids = vim.tbl_map(function(question) return question.id end, state.questions)
  if not vim.deep_equal(ids, previous_ids) then
    render(questions)
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
  render({})
  refresh()

  state.timer = vim.uv.new_timer()
  state.timer:start(0, 150, vim.schedule_wrap(refresh))
end

function M.submit()
  if #state.questions == 0 then
    return
  end
  local lines = vim.api.nvim_buf_get_lines(state.buffer, 0, -1, false)
  local answers = {}
  for question_index = 1, #state.questions do
    local first, last
    for line_index, line in ipairs(lines) do
      if line == ("Answer %d:"):format(question_index) then
        first = line_index + 1
      elseif first and line == ("Question %d:"):format(question_index + 1) then
        last = line_index - 1
        break
      end
    end
    answers[question_index] = first and vim.trim(table.concat(vim.list_slice(lines, first, last), "\n")) or ""
  end
  for index, answer in ipairs(answers) do
    if answer == "" then
      vim.notify(("Answer %d cannot be empty"):format(index), vim.log.levels.WARN)
      return
    end
  end
  vim.fn.mkdir(vim.fs.dirname(state.file), "p")
  local values = {}
  for index, question in ipairs(state.questions) do
    values[index] = vim.json.encode({ type = "answer", question_id = question.id, text = answers[index] })
  end
  vim.fn.writefile(values, state.file, "a")
  if vim.env.HERDR_PANE_ID then
    vim.system({ vim.env.HERDR_BIN_PATH or "herdr", "pane", "close", vim.env.HERDR_PANE_ID }, { detach = true })
  end
  vim.cmd("qa")
end

return M
