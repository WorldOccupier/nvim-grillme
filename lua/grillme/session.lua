local M = {}

function M.read_events(file)
  if vim.fn.filereadable(file) == 0 then
    return {}
  end

  local result = {}
  for _, line in ipairs(vim.fn.readfile(file)) do
    local ok, value = pcall(vim.json.decode, line)
    if ok then
      table.insert(result, value)
    end
  end
  return result
end

function M.pending_questions(file)
  local questions, answered = {}, {}
  for _, value in ipairs(M.read_events(file)) do
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

function M.append_answers(file, questions, answers)
  vim.fn.mkdir(vim.fs.dirname(file), "p")
  local values = {}
  for index, question in ipairs(questions) do
    values[index] = vim.json.encode({
      type = "answer",
      question_id = question.id,
      text = answers[index],
      timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ"),
    })
  end
  vim.fn.writefile(values, file, "a")
end

return M
