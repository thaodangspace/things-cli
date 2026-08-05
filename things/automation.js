// Fixed JXA entrypoint for things-cli.
//
// Keep user-controlled values out of this source. osascript invokes run(argv)
// and supplies the operation name and JSON request as process arguments.

function success(data) {
  return JSON.stringify({ ok: true, data: data });
}

function failure(code, message) {
  return JSON.stringify({ ok: false, error: { code: code, message: message } });
}

function text(value) {
  if (value === null || value === undefined) return "";
  return String(value);
}

function optionalText(value) {
  if (value === null || value === undefined || value === "") return null;
  return text(value);
}

function callValue(object, property) {
  try {
    if (!object || typeof object[property] !== "function") return null;
    return object[property]();
  } catch (error) {
    return null;
  }
}

function objectID(object) {
  var value = callValue(object, "id");
  return optionalText(value);
}

function objectName(object) {
  return text(callValue(object, "name"));
}

function relationRef(object, property) {
  var relation = callValue(object, property);
  if (!relation) return null;
  var id = objectID(relation);
  if (!id) return null;
  return { id: id, title: objectName(relation) };
}

function dateParts(value) {
  if (value === null || value === undefined || value === "") return null;
  try {
    var date = value instanceof Date ? value : new Date(value);
    if (isNaN(date.getTime())) return null;
    return {
      year: date.getFullYear(),
      month: date.getMonth() + 1,
      day: date.getDate(),
      iso: date.toISOString()
    };
  } catch (error) {
    return null;
  }
}

function dateOnly(value) {
  var parts = dateParts(value);
  if (!parts) return null;
  return parts.year + "-" + (parts.month < 10 ? "0" : "") + parts.month + "-" +
    (parts.day < 10 ? "0" : "") + parts.day;
}

function dateTime(value) {
  var parts = dateParts(value);
  return parts ? parts.iso : null;
}

function statusName(value) {
  var status = text(value).toLowerCase();
  if (status === "open" || status === "completed" || status === "canceled") return status;
  if (status === "cancelled") return "canceled";
  return status || "unknown";
}

function taskProperties(task) {
  try {
    return task.properties();
  } catch (error) {
    return {};
  }
}

function className(task, properties) {
  var values = properties || taskProperties(task);
  return text(values.pcls);
}

function isProject(task, facts, properties) {
  return className(task, properties) === "project";
}

function taskTags(task, properties, includeIDs) {
  var output = [];
  var names = text(properties.tagNames);
  if (names) {
    names.split(",").forEach(function (name) {
      var trimmed = name.replace(/^\\s+|\\s+$/g, "");
      if (trimmed) output.push({ id: null, title: trimmed });
    });
  }
  if (!includeIDs) return output;
  try {
    var tags = task.tags();
    output = [];
    for (var i = 0; i < tags.length; i++) {
      var tag = tags[i];
      var id = objectID(tag);
      var name = objectName(tag);
      if (name) output.push({ id: id, title: name });
    }
  } catch (error) {
    // A missing or inaccessible relation is represented by the names above.
  }
  return output;
}

function itemRecord(task, facts, properties, includeTagIDs) {
  var id = objectID(task);
  if (!id) return null;
  var values = properties || taskProperties(task);
  var fact = facts[id] || {};
  var tags = taskTags(task, values, includeTagIDs === true);
  var status = statusName(values.status);
  var completion = values.completionDate;
  if (!completion && status === "canceled") completion = values.cancellationDate;
  var area = values.area ? relationRef(task, "area") : null;
  var project = values.project ? relationRef(task, "project") : null;
  var record = {
    id: id,
    type: isProject(task, facts, values) ? "project" : "to-do",
    status: status,
    title: text(values.name),
    notes: text(values.notes),
    start: fact.start || "unknown",
    start_date: dateOnly(values.activationDate),
    deadline: dateOnly(values.dueDate),
    creation_date: dateTime(values.creationDate),
    modification_date: dateTime(values.modificationDate),
    completion_date: dateTime(completion),
    area: area,
    project: project,
    heading: null,
    tags: tags.map(function (tag) { return tag.title; }),
    checklist: [],
    trashed: fact.trashed === true
  };
  record._tagIDs = tags.map(function (tag) { return tag.id; });
  return record;
}

function listDefinition(kind) {
  var definitions = {
    inbox: { id: "TMInboxListSource", name: "Inbox" },
    today: { id: "TMTodayListSource", name: "Today" },
    upcoming: { id: "TMCalendarListSource", name: "Upcoming" },
    anytime: { id: "TMNextListSource", name: "Anytime" },
    someday: { id: "TMSomedayListSource", name: "Someday" },
    logbook: { id: "TMLogbookListSource", name: "Logbook" },
    trash: { id: "TMTrashListSource", name: "Trash" }
  };
  return definitions[kind] || null;
}

function resolveList(app, kind) {
  var definition = listDefinition(kind);
  if (!definition) return null;
  try {
    var byID = app.lists.byId(definition.id);
    if (byID.exists()) return byID;
  } catch (error) {
    // Fall back to the localized-independent public name below.
  }
  try {
    var byName = app.lists.byName(definition.name);
    if (byName.exists()) return byName;
  } catch (error) {
    return null;
  }
  return null;
}

function listStart(kind) {
  if (kind === "inbox") return "inbox";
  if (kind === "anytime") return "anytime";
  if (kind === "someday") return "someday";
  return "";
}

function ensureFact(facts, id) {
  if (!facts[id]) facts[id] = { lists: [] };
  return facts[id];
}

function markMembership(facts, id, kind) {
  var fact = ensureFact(facts, id);
  if (fact.lists.indexOf(kind) < 0) fact.lists.push(kind);
  var start = listStart(kind);
  if (start && !fact.start) fact.start = start;
  if (kind === "trash") fact.trashed = true;
}

function mergeFacts(facts, id, incoming) {
  var fact = ensureFact(facts, id);
  if (incoming) {
    if (incoming.lists) {
      incoming.lists.forEach(function (kind) {
        if (fact.lists.indexOf(kind) < 0) fact.lists.push(kind);
      });
    }
    if (incoming.start && !fact.start) fact.start = incoming.start;
    if (incoming.trashed) fact.trashed = true;
  }
  return fact;
}

function tasksOf(container) {
  try {
    if (container && typeof container.toDos === "function") return container.toDos();
  } catch (error) {
    return [];
  }
  return [];
}

function collectUniverse(app, includeTagIDs) {
  var tasks = {};
  var order = [];
  var facts = {};
  var properties = {};
  var visitedChildren = {};

  function addTask(task, incoming, depth) {
    if (!task || depth > 20) return;
    var id = objectID(task);
    if (!id) return;
    mergeFacts(facts, id, incoming);
    if (!tasks[id]) {
      tasks[id] = task;
      properties[id] = taskProperties(task);
      order.push(id);
    }
    if (className(task, properties[id]) !== "project" || visitedChildren[id]) return;
    visitedChildren[id] = true;
    var childTasks = tasksOf(task);
    for (var i = 0; i < childTasks.length; i++) {
      addTask(childTasks[i], { lists: incoming && incoming.lists ? incoming.lists : [] }, depth + 1);
    }
  }

  try {
    var topTasks = app.toDos();
    for (var i = 0; i < topTasks.length; i++) addTask(topTasks[i], null, 0);
  } catch (error) {}
  try {
    var projects = app.projects();
    for (var j = 0; j < projects.length; j++) addTask(projects[j], null, 0);
  } catch (error) {}
  try {
    var lists = app.lists();
    for (var k = 0; k < lists.length; k++) {
      var list = lists[k];
      var listID = objectID(list);
      var kind = "";
      for (var name in { inbox: 1, today: 1, upcoming: 1, anytime: 1, someday: 1, logbook: 1, trash: 1 }) {
        var definition = listDefinition(name);
        if (definition && listID === definition.id) kind = name;
      }
      var listTasks = tasksOf(list);
      for (var m = 0; m < listTasks.length; m++) {
        var taskID = objectID(listTasks[m]);
        if (taskID && kind) markMembership(facts, taskID, kind);
        addTask(listTasks[m], null, 0);
      }
    }
  } catch (error) {}
  try {
    var areas = app.areas();
    for (var n = 0; n < areas.length; n++) {
      var areaTasks = tasksOf(areas[n]);
      for (var p = 0; p < areaTasks.length; p++) addTask(areaTasks[p], null, 0);
    }
  } catch (error) {}

  return { tasks: tasks, order: order, facts: facts, properties: properties, includeTagIDs: includeTagIDs === true };
}

function recordsFromUniverse(universe) {
  var output = [];
  for (var i = 0; i < universe.order.length; i++) {
    var id = universe.order[i];
    var record = itemRecord(universe.tasks[id], universe.facts, universe.properties[id], universe.includeTagIDs);
    if (record) output.push(record);
  }
  return output;
}

function exactRelationMatches(relation, value) {
  if (!relation || !value) return false;
  return relation.id === value || relation.title === value;
}

function tagsMatch(record, value) {
  if (!value) return true;
  if (record.tags.indexOf(value) >= 0) return true;
  return record._tagIDs.indexOf(value) >= 0;
}

function listMatch(record, universe, list) {
  if (!list) return true;
  var fact = universe.facts[record.id];
  return !!(fact && fact.lists && fact.lists.indexOf(list) >= 0);
}

function parseQueryDate(value) {
  if (!value) return null;
  var raw = String(value);
  var dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/.exec(raw);
  var date;
  if (dateOnly) {
    date = new Date(Number(dateOnly[1]), Number(dateOnly[2]) - 1, Number(dateOnly[3]), 0, 0, 0, 0);
  } else {
    date = new Date(raw);
  }
  if (isNaN(date.getTime())) throw new Error("invalid query date: " + raw);
  return date;
}

function timestampMatches(value, after, before) {
  if (!after && !before) return true;
  if (!value) return false;
  var date = parseQueryDate(value);
  if (after && date.getTime() < parseQueryDate(after).getTime()) return false;
  if (before && date.getTime() > parseQueryDate(before).getTime()) return false;
  return true;
}

function dateMatches(value, after, before) {
  if (!after && !before) return true;
  if (!value) return false;
  var actual = String(value);
  if (after && actual < String(after)) return false;
  if (before && actual > String(before)) return false;
  return true;
}

function matches(record, universe, request) {
  if (request.status && record.status !== request.status &&
      !(request.status === "canceled" && record.status === "cancelled")) return false;
  if (request.type && request.type !== record.type &&
      !(request.type === "todo" && record.type === "to-do")) return false;
  if (request.tag && !tagsMatch(record, request.tag)) return false;
  if (request.area && !exactRelationMatches(record.area, request.area)) return false;
  if (request.project && !exactRelationMatches(record.project, request.project)) return false;
  if (request.text) {
    var needle = String(request.text).toLowerCase();
    if (record.title.toLowerCase().indexOf(needle) < 0 &&
        record.notes.toLowerCase().indexOf(needle) < 0) return false;
  }
  if (!timestampMatches(record.creation_date, request.created_after, request.created_before)) return false;
  if (!timestampMatches(record.modification_date, request.modified_after, request.modified_before)) return false;
  if (!dateMatches(record.deadline, request.deadline_after, request.deadline_before)) return false;
  if (!dateMatches(record.start_date, request.start_after, request.start_before)) return false;
  if (request.list && !listMatch(record, universe, request.list)) return false;
  return true;
}

function cleanRecord(record) {
  delete record._tagIDs;
  return record;
}

function sortRecords(records) {
  return records.sort(function (a, b) {
    var at = a.title.toLowerCase();
    var bt = b.title.toLowerCase();
    if (at < bt) return -1;
    if (at > bt) return 1;
    return a.id < b.id ? -1 : (a.id > b.id ? 1 : 0);
  });
}

function limited(records, limit, all) {
  if (all === true) return records;
  var n = Number(limit);
  if (!n || n < 1) n = 50;
  return records.slice(0, n);
}

function listRecordsRaw(app, kind, includeTagIDs) {
  var list = resolveList(app, kind);
  if (!list) throw new Error("Things list not found: " + kind);
  var facts = {};
  var tasks = tasksOf(list);
  var output = [];
  for (var i = 0; i < tasks.length; i++) {
    var id = objectID(tasks[i]);
    if (!id) continue;
    markMembership(facts, id, kind);
    var record = itemRecord(tasks[i], facts, taskProperties(tasks[i]), includeTagIDs);
    if (record) output.push(record);
  }
  return { records: output, universe: { facts: facts } };
}

function listRecords(app, kind, limit) {
  var result = listRecordsRaw(app, kind);
  return limited(result.records.map(cleanRecord), limit);
}

function compareQueryValues(a, b) {
  if (a === null || a === undefined || a === "") return (b === null || b === undefined || b === "") ? 0 : 1;
  if (b === null || b === undefined || b === "") return -1;
  if (a < b) return -1;
  if (a > b) return 1;
  return 0;
}

function querySortRecords(records, sort) {
  if (!sort || sort === "native") return records;
  var key = sort;
  return records.sort(function (a, b) {
    var av;
    var bv;
    if (key === "title") {
      av = a.title.toLowerCase();
      bv = b.title.toLowerCase();
    } else if (key === "created") {
      av = a.creation_date;
      bv = b.creation_date;
    } else if (key === "modified") {
      av = a.modification_date;
      bv = b.modification_date;
    } else if (key === "deadline") {
      av = a.deadline;
      bv = b.deadline;
    } else if (key === "start") {
      av = a.start_date;
      bv = b.start_date;
    } else {
      return 0;
    }
    var result = compareQueryValues(av, bv);
    if (result !== 0) return result;
    return a.id < b.id ? -1 : (a.id > b.id ? 1 : 0);
  });
}

function findItem(app, id) {
  var target = null;
  try {
    var todo = app.toDos.byId(id);
    if (todo.exists()) target = todo;
  } catch (error) {}
  if (target) return target;
  try {
    var project = app.projects.byId(id);
    if (project.exists()) target = project;
  } catch (error) {}
  return target;
}

function membershipForItem(app, id) {
  var facts = {};
  var kinds = ["inbox", "today", "upcoming", "anytime", "someday", "logbook", "trash"];
  for (var i = 0; i < kinds.length; i++) {
    var list = resolveList(app, kinds[i]);
    if (!list) continue;
    var tasks = tasksOf(list);
    for (var j = 0; j < tasks.length; j++) {
      if (objectID(tasks[j]) === id) markMembership(facts, id, kinds[i]);
    }
  }
  return facts;
}

function itemRecordWithMembership(app, task) {
  var id = objectID(task);
  var record = itemRecord(task, membershipForItem(app, id), taskProperties(task), false);
  if (!record) return null;
  return cleanRecord(record);
}

function areaRecords(app) {
  var areas = [];
  var source = app.areas();
  for (var i = 0; i < source.length; i++) {
    var id = objectID(source[i]);
    if (id) areas.push({ id: id, title: objectName(source[i]) });
  }
  return sortRecords(areas);
}

function tagRecords(app) {
  var tags = [];
  var source = app.tags();
  for (var i = 0; i < source.length; i++) {
    var id = objectID(source[i]);
    if (id) tags.push({ id: id, title: objectName(source[i]) });
  }
  return sortRecords(tags);
}

function projectRecords(app, request) {
  var projects = app.projects();
  var output = [];
  var facts = {};
  for (var i = 0; i < projects.length; i++) {
    var record = itemRecord(projects[i], facts, taskProperties(projects[i]), false);
    if (!record) continue;
    if (request.area && !exactRelationMatches(record.area, request.area)) continue;
    output.push(cleanRecord(record));
  }
  return limited(sortRecords(output), request.limit);
}

function parseDateValue(value) {
  if (!value) return null;
  var raw = String(value);
  var match = /^(\\d{4})-(\\d{2})-(\\d{2})(?:@(\\d{2}):(\\d{2}))?$/.exec(raw);
  if (!match) throw new Error("invalid date: " + raw);
  var date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]),
    match[4] ? Number(match[4]) : 12, match[5] ? Number(match[5]) : 0, 0, 0);
  if (isNaN(date.getTime())) throw new Error("invalid date: " + raw);
  return date;
}

function relativeDate(value) {
  var now = new Date();
  if (value === "today") return new Date(now.getFullYear(), now.getMonth(), now.getDate(), 12, 0, 0, 0);
  if (value === "tomorrow") return new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 12, 0, 0, 0);
  return null;
}

function resolveListTarget(app, id, name) {
  if (id) {
    try {
      var byID = app.lists.byId(id);
      if (byID.exists()) return byID;
    } catch (error) {}
  }
  if (name) {
    try {
      var byName = app.lists.byName(name);
      if (byName.exists()) return byName;
    } catch (error) {}
  }
  throw new Error("list not found: " + (id || name));
}

function resolveAreaTarget(app, id, name) {
  if (id) {
    try {
      var byID = app.areas.byId(id);
      if (byID.exists()) return byID;
    } catch (error) {}
  }
  if (name) {
    try {
      var byName = app.areas.byName(name);
      if (byName.exists()) return byName;
    } catch (error) {}
  }
  throw new Error("area not found: " + (id || name));
}

function resolveProjectTarget(app, id, name) {
  if (id) {
    try {
      var byID = app.projects.byId(id);
      if (byID.exists()) return byID;
    } catch (error) {}
  }
  if (name) {
    try {
      var byName = app.projects.byName(name);
      if (byName.exists()) return byName;
    } catch (error) {}
  }
  throw new Error("project not found: " + (id || name));
}

function applyWhen(app, task, value) {
  if (!value) return;
  var target = relativeDate(value);
  if (!target && value !== "anytime" && value !== "someday") target = parseDateValue(value);
  if (target) {
    app.schedule(task, { "for": target });
    return;
  }
  if (value === "anytime" || value === "someday") {
    app.move(task, { to: resolveListTarget(app, "", value === "anytime" ? "Anytime" : "Someday") });
    return;
  }
  throw new Error("unsupported schedule: " + value);
}

function applyListDestination(app, task, request) {
  if (request.list || request.list_id) {
    app.move(task, { to: resolveListTarget(app, request.list_id, request.list) });
  }
}

function applyCommonFields(app, task, request) {
  if (request.notes !== undefined) task.notes = request.notes;
  if (request.tags !== undefined) task.tagNames = (request.tags || []).join(",");
  if (request.deadline) task.dueDate = parseDateValue(request.deadline);
  if (request.when) applyWhen(app, task, request.when);
}

function actionResult(action, task) {
  return { action: action, id: objectID(task) };
}

function addTodo(app, request) {
  if (request.completed && request.canceled) throw new Error("completed and canceled cannot both be true");
  if ((request.list || request.list_id) && (request.project || request.project_id)) {
    throw new Error("a todo cannot target both a list and a project");
  }
  var project = null;
  var task;
  if (request.project || request.project_id) {
    project = resolveProjectTarget(app, request.project_id, request.project);
    task = app.ToDo({ name: request.title });
    project.toDos.push(task);
  } else {
    task = app.make({ new: "to do", withProperties: { name: request.title } });
  }
  applyCommonFields(app, task, request);
  if (!project) applyListDestination(app, task, request);
  if (request.completed) task.status = "completed";
  if (request.canceled) task.status = "canceled";
  if (request.reveal) task.show();
  return actionResult("add", task);
}

function addProject(app, request) {
  var project = app.make({ new: "project", withProperties: { name: request.title } });
  applyCommonFields(app, project, request);
  if (request.area || request.area_id) project.area = resolveAreaTarget(app, request.area_id, request.area);
  var children = request.to_dos || [];
  for (var i = 0; i < children.length; i++) {
    var child = app.make({ new: "to do", withProperties: { name: children[i] } });
    try {
      child.project = project;
    } catch (error) {
      app.move(child, { to: project });
    }
  }
  if (request.reveal) project.show();
  return actionResult("add-project", project);
}

function updateTask(app, request) {
  var task = findItem(app, request.id);
  if (!task) throw new Error("item not found: " + request.id);
  if (request.project && className(task) !== "project") throw new Error("item is not a project: " + request.id);
  if (request.title !== undefined) task.name = request.title;
  if (request.notes !== undefined) task.notes = request.notes;
  if (request.prepend_notes !== undefined) task.notes = request.prepend_notes + text(callValue(task, "notes"));
  if (request.append_notes !== undefined) task.notes = text(callValue(task, "notes")) + request.append_notes;
  if (request.tags !== undefined) task.tagNames = (request.tags || []).join(",");
  if (request.add_tags !== undefined) {
    var existing = text(callValue(task, "tagNames"));
    var combined = existing ? existing.split(",") : [];
    (request.add_tags || []).forEach(function (tag) {
      if (combined.indexOf(tag) < 0) combined.push(tag);
    });
    task.tagNames = combined.join(",");
  }
  if (request.deadline !== undefined) task.dueDate = request.deadline ? parseDateValue(request.deadline) : null;
  if (request.when !== undefined) applyWhen(app, task, request.when);
  applyListDestination(app, task, request);
  if (request.completed !== undefined && request.completed) task.status = "completed";
  if (request.completed !== undefined && !request.completed && text(callValue(task, "status")) === "completed") task.status = "open";
  if (request.canceled !== undefined && request.canceled) task.status = "canceled";
  if (request.canceled !== undefined && !request.canceled && text(callValue(task, "status")) === "canceled") task.status = "open";
  return actionResult("update", task);
}

function showTarget(app, target) {
  var item = findItem(app, target);
  if (item) {
    item.show();
    return { action: "show", id: target };
  }
  var list = resolveListTarget(app, target, target);
  list.show();
  return { action: "show", id: target };
}

function encodeQuery(value) {
  return encodeURIComponent(String(value)).replace(/[!'()*]/g, function (character) {
    return "%" + character.charCodeAt(0).toString(16).toUpperCase();
  });
}

function searchTarget(query) {
  var current = Application.currentApplication();
  current.includeStandardAdditions = true;
  current.openLocation("things:///search?query=" + encodeQuery(query));
  return { action: "search" };
}

function dispatch(operation, request) {
  if (operation === "health") return { operation: operation, request: request };
  var app = Application("com.culturedcode.ThingsMac");
  if (operation === "list") return listRecords(app, request.list, request.limit);
  if (operation === "query") {
    if (request.list) {
      var nativeList = listRecordsRaw(app, request.list, !!request.tag);
      var nativeRecords = nativeList.records.filter(function (record) {
        return matches(record, nativeList.universe, request);
      });
      var nativeOrdered = querySortRecords(nativeRecords, request.sort);
      if (request.reverse === true) nativeOrdered.reverse();
      return limited(nativeOrdered.map(cleanRecord), request.limit, request.all);
    }
    var universe = collectUniverse(app, !!request.tag);
    var records = recordsFromUniverse(universe).filter(function (record) {
      return matches(record, universe, request);
    });
    var ordered = querySortRecords(records, request.sort || "title");
    if (request.reverse === true) ordered.reverse();
    return limited(ordered, request.limit, request.all).map(cleanRecord);
  }
  if (operation === "get") {
    var task = findItem(app, request.id);
    if (!task) throw new Error("item not found: " + request.id);
    return itemRecordWithMembership(app, task);
  }
  if (operation === "list-projects") return projectRecords(app, request);
  if (operation === "list-areas") return areaRecords(app);
  if (operation === "list-tags") return tagRecords(app);
  if (operation === "add") return addTodo(app, request);
  if (operation === "add-project") return addProject(app, request);
  if (operation === "update") return updateTask(app, request);
  if (operation === "complete") {
    var completeTask = findItem(app, request.id);
    if (!completeTask) throw new Error("item not found: " + request.id);
    completeTask.status = "completed";
    return actionResult("complete", completeTask);
  }
  if (operation === "cancel") {
    var cancelTask = findItem(app, request.id);
    if (!cancelTask) throw new Error("item not found: " + request.id);
    cancelTask.status = "canceled";
    return actionResult("cancel", cancelTask);
  }
  if (operation === "show") return showTarget(app, request.target);
  if (operation === "search") return searchTarget(request.query);
  throw new Error("unsupported operation: " + operation);
}

function run(argv) {
  if (!argv || argv.length < 2) {
    return failure("invalid_arguments", "operation and JSON request are required");
  }

  var operation = argv[0];
  var request;
  try {
    request = JSON.parse(argv[1]);
  } catch (error) {
    return failure("invalid_request", "request is not valid JSON");
  }

  try {
    return success(dispatch(operation, request));
  } catch (error) {
    var message = text(error);
    if (message.indexOf("Error: item not found") === 0 || message.indexOf("item not found") === 0) {
      return failure("not_found", message);
    }
    return failure("application_error", message);
  }
}
