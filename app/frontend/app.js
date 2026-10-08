const API_URL = "";

const taskForm = document.getElementById("task-form");
const taskTitleInput = document.getElementById("task-title");

const tasksContainer = document.getElementById("tasks");

const loadingElement = document.getElementById("loading");
const emptyState = document.getElementById("empty-state");

const errorMessage = document.getElementById("error-message");

const refreshButton = document.getElementById("refresh-button");

const apiStatus = document.getElementById("api-status");

async function checkApiHealth() {
  try {
    const response = await fetch(`${API_URL}/health`);

    if (!response.ok) {
      throw new Error("API unavailable");
    }

    apiStatus.textContent = "API Online";

    apiStatus.className = "status status-online";
  } catch (error) {
    apiStatus.textContent = "API Offline";

    apiStatus.className = "status status-offline";
  }
}

async function loadTasks() {
  hideError();

  loadingElement.classList.remove("hidden");

  emptyState.classList.add("hidden");

  tasksContainer.innerHTML = "";

  try {
    const response = await fetch(`${API_URL}/api/tasks`);

    if (!response.ok) {
      throw new Error(`Server returned ${response.status}`);
    }

    const tasks = await response.json();

    renderTasks(tasks);
  } catch (error) {
    showError(`Unable to load tasks: ${error.message}`);
  } finally {
    loadingElement.classList.add("hidden");
  }
}

function renderTasks(tasks) {
  tasksContainer.innerHTML = "";

  if (tasks.length === 0) {
    emptyState.classList.remove("hidden");

    return;
  }

  emptyState.classList.add("hidden");

  tasks.forEach((task) => {
    const element = createTaskElement(task);

    tasksContainer.appendChild(element);
  });
}

function createTaskElement(task) {
  const taskElement = document.createElement("div");

  taskElement.className = "task";

  const left = document.createElement("div");

  left.className = "task-left";

  const checkbox = document.createElement("input");

  checkbox.type = "checkbox";

  checkbox.checked = task.completed;

  const title = document.createElement("span");

  title.textContent = task.title;

  title.className = task.completed ? "task-title task-completed" : "task-title";

  checkbox.addEventListener("change", async () => {
    await updateTask(task.id, task.title, checkbox.checked);
  });

  const deleteButton = document.createElement("button");

  deleteButton.textContent = "Delete";

  deleteButton.className = "delete-button";

  deleteButton.addEventListener("click", async () => {
    await deleteTask(task.id);
  });

  left.appendChild(checkbox);

  left.appendChild(title);

  taskElement.appendChild(left);

  taskElement.appendChild(deleteButton);

  return taskElement;
}

async function createTask(title) {
  hideError();

  try {
    const response = await fetch(`${API_URL}/api/tasks`, {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        title: title,
      }),
    });

    if (!response.ok) {
      throw new Error(`Server returned ${response.status}`);
    }

    await loadTasks();
  } catch (error) {
    showError(`Unable to create task: ${error.message}`);
  }
}

async function updateTask(id, title, completed) {
  hideError();

  try {
    const response = await fetch(`${API_URL}/api/tasks/${id}`, {
      method: "PUT",

      headers: {
        "Content-Type": "application/json",
      },

      body: JSON.stringify({
        title,
        completed,
      }),
    });

    if (!response.ok) {
      throw new Error(`Server returned ${response.status}`);
    }

    await loadTasks();
  } catch (error) {
    showError(`Unable to update task: ${error.message}`);

    await loadTasks();
  }
}

async function deleteTask(id) {
  hideError();

  try {
    const response = await fetch(`${API_URL}/api/tasks/${id}`, {
      method: "DELETE",
    });

    if (!response.ok) {
      throw new Error(`Server returned ${response.status}`);
    }

    await loadTasks();
  } catch (error) {
    showError(`Unable to delete task: ${error.message}`);
  }
}

function showError(message) {
  errorMessage.textContent = message;

  errorMessage.classList.remove("hidden");
}

function hideError() {
  errorMessage.classList.add("hidden");
}

taskForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const title = taskTitleInput.value.trim();

  if (!title) {
    return;
  }

  await createTask(title);

  taskTitleInput.value = "";

  taskTitleInput.focus();
});

refreshButton.addEventListener("click", loadTasks);

checkApiHealth();

loadTasks();
