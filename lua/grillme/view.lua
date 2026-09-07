local M = {}

local recommendation_namespace = vim.api.nvim_create_namespace("grillme_recommendations")

function M.render(buffer, questions)
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

  vim.api.nvim_buf_set_lines(buffer, 0, -1, false, lines)
  vim.api.nvim_buf_clear_namespace(buffer, recommendation_namespace, 0, -1)
  local question_index = 1
  for line_index, line in ipairs(lines) do
    if line == "```text" then
      local recommendation = questions[question_index].recommended_answer
      if recommendation and recommendation ~= "" then
        for offset, recommendation_line in ipairs(vim.split(recommendation, "\n", { plain = true })) do
          vim.api.nvim_buf_set_extmark(buffer, recommendation_namespace, line_index + offset - 1, 0, {
            end_col = #recommendation_line,
            hl_group = "Comment",
          })
        end
      end
      question_index = question_index + 1
    end
  end
  vim.bo[buffer].modified = false

  local first_answer_line
  if #questions > 0 then
    for line_index, line in ipairs(lines) do
      if line == "```text" then
        first_answer_line = line_index + 1
        break
      end
    end
  end
  return first_answer_line
end

function M.extract_answers(lines, question_count)
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
  for index = 1, question_count do
    answers[index] = answers[index] or ""
  end
  return answers
end

return M
