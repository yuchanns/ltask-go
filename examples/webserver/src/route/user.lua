local json = require("json")

local S = {}

local users = {}

function S.get(path, query, header, body)
  local name = query.name or "anonymous"
  local user = users[name]
  if not user then
    return 404, ('{"message": "User %s not found"}'):format(name), { ["Content-Type"] = "application/json" }
  end
  local data = json.encode({
    name = user.name,
    age = user.age,
  })
  return 200, data, { ["Content-Type"] = "application/json" }
end

function S.post(path, query, header, body)
  body = json.decode(body)
  local user = {
    name = body.name or "anonymous",
    age = tonumber(body.age or 24),
  }
  users[user.name] = user
  local data = json.encode({
    message = ("User %s created"):format(user.name),
  })
  return 201, data, { ["Content-Type"] = "application/json" }
end

return S
