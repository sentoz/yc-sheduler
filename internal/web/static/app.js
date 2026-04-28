const state = {
  currentMonth: startOfMonth(new Date()),
  selectedDate: null,
  selectedFilter: "all",
  events: [],
};

const monthTitle = document.getElementById("month-title");
const timezoneLabel = document.getElementById("timezone-label");
const calendarGrid = document.getElementById("calendar-grid");
const selectedDateTitle = document.getElementById("selected-date-title");
const selectedDateCount = document.getElementById("selected-date-count");
const selectedDayEvents = document.getElementById("selected-day-events");
const actionFilter = document.getElementById("action-filter");

document.getElementById("prev-month").addEventListener("click", () => {
  state.currentMonth = startOfMonth(new Date(state.currentMonth.getFullYear(), state.currentMonth.getMonth() - 1, 1));
  loadMonth();
});

document.getElementById("next-month").addEventListener("click", () => {
  state.currentMonth = startOfMonth(new Date(state.currentMonth.getFullYear(), state.currentMonth.getMonth() + 1, 1));
  loadMonth();
});

document.getElementById("today").addEventListener("click", () => {
  state.currentMonth = startOfMonth(new Date());
  state.selectedDate = formatDateLocal(new Date());
  loadMonth();
});

actionFilter.addEventListener("change", () => {
  state.selectedFilter = actionFilter.value;
  render();
});

loadMonth();

async function loadMonth() {
  const from = formatDateLocal(startOfMonth(state.currentMonth));
  const to = formatDateLocal(endOfMonth(state.currentMonth));

  try {
    const response = await fetch(`/api/calendar?from=${from}&to=${to}`);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const payload = await response.json();
    state.events = payload.events || [];
    state.selectedDate = normalizeSelectedDate(payload.events, from);

    monthTitle.textContent = payload.title || formatMonth(state.currentMonth);
    timezoneLabel.textContent = payload.timezone || "";
    render();
  } catch (error) {
    monthTitle.textContent = formatMonth(state.currentMonth);
    timezoneLabel.textContent = "Не удалось загрузить календарь";
    calendarGrid.innerHTML = "";
    selectedDateTitle.textContent = "Ошибка";
    selectedDateCount.textContent = "";
    selectedDayEvents.className = "event-list empty-state";
    selectedDayEvents.textContent = "Календарь временно недоступен.";
    console.error(error);
  }
}

function render() {
  renderCalendar();
  renderSelectedDay();
}

function renderCalendar() {
  const cells = buildCalendarCells(state.currentMonth);
  const today = formatDateLocal(new Date());
  const grouped = groupEventsByDate(filteredEvents());

  calendarGrid.innerHTML = "";

  cells.forEach((cell) => {
    const events = grouped[cell.date] || [];
    const day = document.createElement("button");
    day.type = "button";
    day.className = "calendar-day";
    day.dataset.date = cell.date;

    if (!cell.inCurrentMonth) {
      day.classList.add("calendar-day--muted");
    }
    if (cell.date === today) {
      day.classList.add("calendar-day--today");
    }
    if (cell.date === state.selectedDate) {
      day.classList.add("calendar-day--selected");
    }

    const head = document.createElement("div");
    head.className = "calendar-day__head";
    head.innerHTML = `
      <span class="calendar-day__date">${cell.dayNumber}</span>
      <span class="calendar-day__count">${events.length > 0 ? `${events.length} шт.` : ""}</span>
    `;
    day.appendChild(head);

    const eventsWrap = document.createElement("div");
    eventsWrap.className = "calendar-day__events";

    events.slice(0, 3).forEach((event) => {
      const chip = document.createElement("div");
      chip.className = `event-chip event-chip--${event.action}`;
      chip.innerHTML = `
        <span class="event-chip__time">${shortTime(event.local_time)}</span>
        <span>${event.action}</span>
        <span class="event-chip__meta">${event.schedule_name} · ${event.resource_type} · ${formatStateLabel(event)}</span>
      `;
      eventsWrap.appendChild(chip);
    });

    if (events.length > 3) {
      const more = document.createElement("div");
      more.className = "event-chip__meta";
      more.textContent = `Еще ${events.length - 3}`;
      eventsWrap.appendChild(more);
    }

    day.appendChild(eventsWrap);
    day.addEventListener("click", () => {
      state.selectedDate = cell.date;
      render();
    });
    calendarGrid.appendChild(day);
  });
}

function renderSelectedDay() {
  const events = filteredEvents().filter((event) => event.local_date === state.selectedDate);

  selectedDateTitle.textContent = state.selectedDate ? formatHumanDate(state.selectedDate) : "Выберите день";
  selectedDateCount.textContent = `${events.length} событий`;

  if (!state.selectedDate || events.length === 0) {
    selectedDayEvents.className = "event-list empty-state";
    selectedDayEvents.textContent = state.selectedDate ? "На этот день событий нет." : "Выберите день в календаре.";
    return;
  }

  selectedDayEvents.className = "event-list";
  selectedDayEvents.innerHTML = "";

  events.forEach((event) => {
    const row = document.createElement("article");
    row.className = "event-row";
    row.innerHTML = `
      <div class="event-row__top">
        <span class="event-row__time">${shortTime(event.local_time)}</span>
        <span class="event-row__action">${event.action}</span>
      </div>
      <h3 class="event-row__title">${event.schedule_name}</h3>
      <p class="event-row__meta">${event.resource_type} · ${event.resource_id}</p>
      <p class="event-row__meta">${event.folder_id || ""}</p>
      <div class="event-row__status">
        <span class="status-badge ${statusBadgeClass(event)}">${formatStateLabel(event)}</span>
        <span class="event-row__status-text">${statusDescription(event)}</span>
      </div>
    `;
    selectedDayEvents.appendChild(row);
  });
}

function filteredEvents() {
  if (state.selectedFilter === "all") {
    return state.events;
  }
  return state.events.filter((event) => event.action === state.selectedFilter);
}

function groupEventsByDate(events) {
  return events.reduce((acc, event) => {
    acc[event.local_date] = acc[event.local_date] || [];
    acc[event.local_date].push(event);
    return acc;
  }, {});
}

function buildCalendarCells(monthDate) {
  const first = startOfMonth(monthDate);
  const last = endOfMonth(monthDate);
  const startOffset = (first.getDay() + 6) % 7;
  const start = new Date(first);
  start.setDate(first.getDate() - startOffset);

  const endOffset = 6 - ((last.getDay() + 6) % 7);
  const end = new Date(last);
  end.setDate(last.getDate() + endOffset);

  const cells = [];
  for (let cursor = new Date(start); cursor <= end; cursor.setDate(cursor.getDate() + 1)) {
    cells.push({
      date: formatDateLocal(cursor),
      dayNumber: cursor.getDate(),
      inCurrentMonth: cursor.getMonth() === monthDate.getMonth(),
    });
  }
  return cells;
}

function normalizeSelectedDate(events, fallbackDate) {
  if (state.selectedDate) {
    return state.selectedDate;
  }
  if (events.length > 0) {
    return events[0].local_date;
  }
  return fallbackDate;
}

function startOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

function endOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}

function formatDateLocal(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function formatMonth(date) {
  return new Intl.DateTimeFormat("ru-RU", { month: "long", year: "numeric" }).format(date);
}

function formatHumanDate(dateString) {
  const [year, month, day] = dateString.split("-").map(Number);
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(new Date(year, month - 1, day));
}

function shortTime(value) {
  return value.slice(0, 5);
}

function formatStateLabel(event) {
  if (event.status_error || !event.state) {
    return "unknown";
  }
  return event.state.toLowerCase();
}

function statusDescription(event) {
  if (event.status_error) {
    return event.status_error;
  }
  if (!event.state) {
    return "Статус недоступен";
  }
  if (event.transitional) {
    return "Переходное состояние";
  }
  return "Стабильное состояние";
}

function statusBadgeClass(event) {
  if (event.status_error || !event.state) {
    return "status-badge--unknown";
  }
  if (event.transitional) {
    return "status-badge--transitional";
  }
  if (event.state.toLowerCase() === "running") {
    return "status-badge--running";
  }
  if (event.state.toLowerCase() === "stopped") {
    return "status-badge--stopped";
  }
  return "status-badge--unknown";
}
