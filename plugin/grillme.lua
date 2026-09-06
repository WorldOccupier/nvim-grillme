vim.api.nvim_create_user_command("GrillMeOpen", function()
  require("grillme").open()
end, {})

vim.api.nvim_create_user_command("GrillMeSubmit", function()
  require("grillme").submit()
end, {})
