local M = {}

local recommendation_namespace = vim.api.nvim_create_namespace("grillme_recommendations")

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
  local lines = { "# GrillMe", "", "_Write each answer below its heading, then press `<C-s>` to submit._", "" }
  if #questions == 0 then
    table.insert(lines, "> Waiting for a question…")
  end
  for index, question in ipairs(questions) do
    local question_lines = vim.split(question.text, "\n", { plain = true })
    table.insert(lines, "## " .. question_lines[1])
    if #question_lines > 1 then
      vim.list_extend(lines, vim.list_slice(question_lines, 2))
    end
    vim.list_extend(lines, { "", "**Your answer:**", "```text" })
    if question.recommended_answer and question.recommended_answer ~= "" then
      vim.list_extend(lines, vim.split(question.recommended_answer, "\n", { plain = true }))
    else
      table.insert(lines, "")
    end
    table.insert(lines, "```")
    if index < #questions then
      vim.list_extend(lines, { "", "---", "" })
    end
  end
  vim.api.nvim_buf_set_lines(state.buffer, 0, -1, false, lines)
  vim.api.nvim_buf_clear_namespace(state.buffer, recommendation_namespace, 0, -1)
  local question_index = 1
  for line_index, line in ipairs(lines) do
    if line == "```text" then
      local recommendation = questions[question_index].recommended_answer
      if recommendation and recommendation ~= "" then
        local recommendation_lines = vim.split(recommendation, "\n", { plain = true })
        for offset, recommendation_line in ipairs(recommendation_lines) do
          vim.api.nvim_buf_set_extmark(state.buffer, recommendation_namespace, line_index + offset - 1, 0, {
            end_col = #recommendation_line,
            hl_group = "Comment",
          })
        end
      end
      question_index = question_index + 1
    end
  end
  vim.bo[state.buffer].modified = false
  if #questions > 0 then
    local win = vim.fn.bufwinid(state.buffer)
    if win ~= -1 then
      local first_answer_line = 1
      for line_index, line in ipairs(lines) do
        if line == "```text" then
          first_answer_line = line_index + 1
          break
        end
      end
      vim.api.nvim_win_set_cursor(win, { first_answer_line, 0 })
      vim.api.nvim_set_current_win(win)
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

local function stop_timer(timer)
  if not timer or timer:is_closing() then
    return
  end
  timer:stop()
  timer:close()
  if state.timer == timer then
    state.timer = nil
  end
end

local function close_session(buffer, timer)
  stop_timer(timer)
  if state.buffer == buffer then
    state.buffer = nil
    state.questions = {}
  end
end

function M.open()
  if state.buffer and vim.api.nvim_buf_is_valid(state.buffer) then
    local win = vim.fn.bufwinid(state.buffer)
    if win ~= -1 then
      vim.api.nvim_set_current_win(win)
    else
      vim.api.nvim_set_current_buf(state.buffer)
    end
    return
  end

  stop_timer(state.timer)
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

  local buffer = state.buffer
  local timer = vim.uv.new_timer()
  state.timer = timer

  vim.api.nvim_create_autocmd({ "BufDelete", "BufWipeout" }, {
    buffer = buffer,
    once = true,
    callback = function()
      close_session(buffer, timer)
    end,
  })
  vim.api.nvim_create_autocmd("VimLeavePre", {
    once = true,
    callback = function()
      close_session(buffer, timer)
    end,
  })

  timer:start(0, 150, function()
    vim.schedule(function()
      if state.buffer ~= buffer or state.timer ~= timer or timer:is_closing()
          or not vim.api.nvim_buf_is_valid(buffer) then
        return
      end
      refresh()
    end)
  end)
end

function M.submit()
  if #state.questions == 0 then
    return
  end
  local lines = vim.api.nvim_buf_get_lines(state.buffer, 0, -1, false)
  local answers = {}
  local question_index = 1
  local waiting_for_fence = false
  local first
  for line_index, line in ipairs(lines) do
    if not first and line == "**Your answer:**" then
      waiting_for_fence = true
    elseif waiting_for_fence and line == "```text" then
      first = line_index + 1
      waiting_for_fence = false
    elseif first and line == "```" then
      answers[question_index] = vim.trim(table.concat(vim.list_slice(lines, first, line_index - 1), "\n"))
      question_index = question_index + 1
      first = nil
    end
  end
  for index = 1, #state.questions do
    local answer = answers[index] or ""
    answers[index] = answer
    if answer == "" then
      vim.notify(("Answer %d cannot be empty"):format(index), vim.log.levels.WARN)
      return
    end
  end
  vim.fn.mkdir(vim.fs.dirname(state.file), "p")
  local values = {}
  for index, question in ipairs(state.questions) do
    values[index] = vim.json.encode({
      type = "answer",
      question_id = question.id,
      text = answers[index],
      timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ"),
    })
  end
  vim.fn.writefile(values, state.file, "a")
  close_session(state.buffer, state.timer)
  if vim.env.HERDR_PANE_ID then
    vim.system({ vim.env.HERDR_BIN_PATH or "herdr", "pane", "close", vim.env.HERDR_PANE_ID }, { detach = true })
  end
  vim.cmd("qa")
end

return M
