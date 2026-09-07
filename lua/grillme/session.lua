local M = {}

local function lock_is_stale(lock_path)
  local owner = io.open(vim.fs.joinpath(lock_path, "owner"), "r")
  if not owner then
    local stat = vim.uv.fs_stat(lock_path)
    return stat and os.time() - stat.mtime.sec > 2
  end
  local pid = tonumber(owner:read("*l"))
  owner:close()
  if not pid then
    local stat = vim.uv.fs_stat(lock_path)
    return stat and os.time() - stat.mtime.sec > 2
  end
  return vim.uv.kill(pid, 0) == nil
end

local function acquire_lock(file)
  local lock_path = file .. ".lock"
  vim.fn.mkdir(vim.fs.dirname(file), "p")
  local acquired = vim.wait(10000, function()
    if vim.uv.fs_mkdir(lock_path, 448) then
      local owner, err = io.open(vim.fs.joinpath(lock_path, "owner"), "w")
      if not owner then
        vim.uv.fs_rmdir(lock_path)
        error("write session lock owner: " .. err)
      end
      owner:write(vim.uv.os_getpid(), "\n")
      owner:close()
      return true
    end
    if lock_is_stale(lock_path) then
      os.remove(vim.fs.joinpath(lock_path, "owner"))
      vim.uv.fs_rmdir(lock_path)
    end
    return false
  end, 10)
  if not acquired then
    error("timed out waiting for session lock " .. lock_path)
  end
  return function()
    os.remove(vim.fs.joinpath(lock_path, "owner"))
    local ok, err = vim.uv.fs_rmdir(lock_path)
    if not ok then
      error("release session lock: " .. tostring(err))
    end
  end
end

local function with_lock(file, mutation)
  local release = acquire_lock(file)
  local ok, mutation_err = xpcall(mutation, debug.traceback)
  local released, release_err = pcall(release)
  if not ok then
    error(mutation_err)
  end
  if not released then
    error(release_err)
  end
end

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
  with_lock(file, function()
    local output, err = io.open(file, "a")
    if not output then
      error("open session file: " .. err)
    end
    for index, question in ipairs(questions) do
      local value = vim.json.encode({
        type = "answer",
        question_id = question.id,
        text = answers[index],
        timestamp = os.date("!%Y-%m-%dT%H:%M:%SZ"),
      })
      local ok, write_err = output:write(value, "\n")
      if not ok then
        output:close()
        error("append answer: " .. write_err)
      end
    end
    local ok, close_err = output:close()
    if not ok then
      error("close session file: " .. close_err)
    end
  end)
end

return M
