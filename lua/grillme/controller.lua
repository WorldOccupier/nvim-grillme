local session = require("grillme.session")
local view = require("grillme.view")

local M = {}

local state = {
  file = vim.fs.joinpath(vim.fn.getcwd(), ".grillme", "session.jsonl"),
  questions = {},
  buffer = nil,
  timer = nil,
}

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

local function render(questions)
  if not state.buffer or not vim.api.nvim_buf_is_valid(state.buffer) then
    return
  end
  state.questions = questions
  local first_answer_line = view.render(state.buffer, questions)
  local win = vim.fn.bufwinid(state.buffer)
  if first_answer_line and win ~= -1 then
    vim.api.nvim_win_set_cursor(win, { first_answer_line, 0 })
    vim.api.nvim_set_current_win(win)
  end
end

local function refresh()
  local questions = session.pending_questions(state.file)
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
    callback = function() close_session(buffer, timer) end,
  })
  vim.api.nvim_create_autocmd("VimLeavePre", {
    once = true,
    callback = function() close_session(buffer, timer) end,
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
  local answers = view.extract_answers(lines, #state.questions)
  for index, answer in ipairs(answers) do
    if answer == "" then
      vim.notify(("Answer %d cannot be empty"):format(index), vim.log.levels.WARN)
      return
    end
  end

  session.append_answers(state.file, state.questions, answers)
  close_session(state.buffer, state.timer)
  if vim.env.HERDR_PANE_ID then
    vim.system({ vim.env.HERDR_BIN_PATH or "herdr", "pane", "close", vim.env.HERDR_PANE_ID }, { detach = true })
  end
  vim.cmd("qa")
end

return M
