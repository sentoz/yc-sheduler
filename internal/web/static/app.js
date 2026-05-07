const state = {
  currentWeekStart: startOfWeek(new Date()),
  selectedDate: null,
  selectedHour: null,
  selectedTime: null,
  selectedFilter: "all",
  selectedType: "all",
  selectedSchedule: "",
  events: [],
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
};

const monthTitle = document.getElementById("month-title");
const timezoneLabel = document.getElementById("timezone-label");
const calendarGrid = document.getElementById("calendar-grid");
const selectedDateTitle = document.getElementById("selected-date-title");
const selectedDateCount = document.getElementById("selected-date-count");
const selectedDayEvents = document.getElementById("selected-day-events");
const actionFilter = document.getElementById("action-filter");
const typeFilter = document.getElementById("type-filter");
const scheduleFilter = document.getElementById("schedule-filter");
const scheduleOptions = document.getElementById("schedule-options");
const scheduleFilterIsInput = scheduleFilter && scheduleFilter.tagName === "INPUT";

document.getElementById("prev-month").addEventListener("click", () => {
  state.currentWeekStart = addDays(state.currentWeekStart, -7);
  state.selectedHour = null;
  state.selectedTime = null;
  loadWeek();
});

document.getElementById("next-month").addEventListener("click", () => {
  state.currentWeekStart = addDays(state.currentWeekStart, 7);
  state.selectedHour = null;
  state.selectedTime = null;
  loadWeek();
});

document.getElementById("today").addEventListener("click", () => {
  const today = todayInTimezone();
  state.currentWeekStart = startOfWeek(today);
  state.selectedDate = formatDateLocal(today);
  state.selectedHour = null;
  state.selectedTime = null;
  loadWeek();
});

actionFilter.addEventListener("change", () => {
  state.selectedFilter = actionFilter.value;
  syncFilterControls();
  render();
});

typeFilter.addEventListener("change", () => {
  state.selectedType = typeFilter.value;
  syncFilterControls();
  render();
});

if (scheduleFilter) {
  const scheduleEvent = scheduleFilterIsInput ? "input" : "change";
  scheduleFilter.addEventListener(scheduleEvent, () => {
    state.selectedSchedule = normalizeScheduleFilterValue(scheduleFilter.value);
    syncFilterControls();
    render();
  });
}

loadWeek();

async function loadWeek() {
  const from = formatDateLocal(state.currentWeekStart);
  const to = formatDateLocal(addDays(state.currentWeekStart, 6));

  try {
    const response = await fetch(`/api/calendar?from=${from}&to=${to}`);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const payload = await response.json();
    state.events = payload.events || [];
    state.timezone = payload.timezone || state.timezone;
    syncFilterControls();
    state.selectedDate = normalizeSelectedDate(state.events, from, to);

    monthTitle.textContent = formatWeekTitle(state.currentWeekStart);
    timezoneLabel.textContent = payload.timezone || "";
    render();
  } catch (error) {
    monthTitle.textContent = formatWeekTitle(state.currentWeekStart);
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
  const days = buildWeekDays(state.currentWeekStart);
  const today = formatDateLocal(todayInTimezone());
  const groupedByDate = groupEventsByDate(filteredEvents());

  calendarGrid.innerHTML = "";
  calendarGrid.className = "week-timeline";

  const headerRow = document.createElement("div");
  headerRow.className = "week-timeline__row week-timeline__row--header";

  const corner = document.createElement("div");
  corner.className = "week-timeline__corner";
  headerRow.appendChild(corner);

  days.forEach((day) => {
    const events = groupedByDate[day.date] || [];
    const header = document.createElement("button");
    header.type = "button";
    header.className = "week-day-header";
    header.dataset.date = day.date;
    if (day.date === today) {
      header.classList.add("week-day-header--today");
    }
    if (day.date === state.selectedDate) {
      header.classList.add("week-day-header--selected");
    }
    header.innerHTML = `
      <span class="week-day-header__weekday">${day.weekday}</span>
      <span class="week-day-header__date">${day.dayMonth}</span>
      <span class="week-day-header__count">${events.length}</span>
    `;
    header.addEventListener("click", () => {
      state.selectedDate = day.date;
      state.selectedHour = null;
      state.selectedTime = null;
      render();
    });
    headerRow.appendChild(header);
  });
  calendarGrid.appendChild(headerRow);

  for (let hour = 0; hour < 24; hour++) {
    const hourRow = document.createElement("div");
    hourRow.className = "week-timeline__row";

    const hourLabel = document.createElement("div");
    hourLabel.className = "week-hour-label";
    hourLabel.textContent = `${String(hour).padStart(2, "0")}:00`;
    hourRow.appendChild(hourLabel);

    days.forEach((day) => {
      const dayEvents = groupedByDate[day.date] || [];
      const hourGroups = groupEventsByHour(dayEvents, hour);
      const cell = document.createElement("button");
      cell.type = "button";
      cell.className = "week-hour-cell";
      cell.dataset.date = day.date;
      if (day.date === today) {
        cell.classList.add("week-hour-cell--today");
      }
      if (day.date === state.selectedDate && state.selectedHour === hour) {
        cell.classList.add("week-hour-cell--selected");
      }
      if (hourGroups.length > 0) {
        cell.classList.add("week-hour-cell--filled");
        cell.title = formatHourTooltip(hourGroups);
        cell.setAttribute("aria-label", formatHourTooltip(hourGroups));
      }

      hourGroups.forEach((group) => {
        const bucket = document.createElement("span");
        bucket.className = "time-bucket";
        if (day.date === state.selectedDate && state.selectedTime === group.time) {
          bucket.classList.add("time-bucket--selected");
        }
        bucket.title = formatGroupTooltip(group);
        bucket.setAttribute("aria-label", formatGroupTooltip(group));
        bucket.addEventListener("click", (event) => {
          event.stopPropagation();
          state.selectedDate = day.date;
          state.selectedHour = hour;
          state.selectedTime = group.time;
          render();
        });

        const dots = document.createElement("span");
        dots.className = "time-bucket__dots";

        group.events.slice(0, 12).forEach((event) => {
          const dot = document.createElement("span");
          dot.className = `event-dot event-dot--${event.action}`;
          dot.style.backgroundColor = eventColor(event);
          dot.title = formatEventTooltip(event);
          dots.appendChild(dot);
        });

        bucket.appendChild(dots);

        const count = document.createElement("span");
        count.className = "time-bucket__count";
        count.textContent = `(${group.events.length})`;
        bucket.appendChild(count);

        cell.appendChild(bucket);
      });

      cell.addEventListener("click", () => {
        state.selectedDate = day.date;
        state.selectedHour = hour;
        state.selectedTime = null;
        render();
      });
      hourRow.appendChild(cell);
    });

    calendarGrid.appendChild(hourRow);
  }
}

function renderSelectedDay() {
  const events = filteredEvents().filter((event) => {
    if (event.local_date !== state.selectedDate) {
      return false;
    }
    if (state.selectedHour === null) {
      return true;
    }
    if (state.selectedTime !== null) {
      return event.local_time === state.selectedTime;
    }
    return Number(event.local_time.slice(0, 2)) === state.selectedHour;
  });
  const timeGroups = groupEventsByTime(events);

  selectedDateTitle.textContent = selectedDetailsTitle();
  selectedDateCount.textContent = `${events.length} событий`;

  if (!state.selectedDate || events.length === 0) {
    selectedDayEvents.className = "event-list empty-state";
    selectedDayEvents.textContent = emptyDetailsText();
    return;
  }

  selectedDayEvents.className = "event-list";
  selectedDayEvents.innerHTML = "";

  timeGroups.forEach((group) => {
    const groupCard = document.createElement("article");
    groupCard.className = "event-group";
    groupCard.innerHTML = `
      <div class="event-group__top">
        <span class="event-group__time">${shortTime(group.time)}</span>
        <span class="event-group__count">${group.events.length} шт.</span>
      </div>
    `;

    group.events.forEach((event) => {
      const row = document.createElement("div");
      row.className = "event-row";
      row.innerHTML = `
        <div class="event-row__top">
          <span class="event-row__action">${event.action}</span>
          <span class="status-badge ${statusBadgeClass(event)}">${formatStateLabel(event)}</span>
        </div>
        <h3 class="event-row__title">${displayName(event)}</h3>
        <p class="event-row__meta">${event.resource_type} · ${event.resource_id}</p>
        <p class="event-row__meta">${event.schedule_name}</p>
        <p class="event-row__meta">${event.folder_id || ""}</p>
        <div class="event-row__status">
          <span class="event-row__status-text">${statusDescription(event)}</span>
        </div>
      `;
      groupCard.appendChild(row);
    });

    selectedDayEvents.appendChild(groupCard);
  });
}

function filteredEvents() {
  return applyFilters(state.events);
}

function syncFilterControls() {
  syncActionFilterOptions();
  syncTypeFilterOptions();
  syncScheduleFilterOptions();
}

function syncActionFilterOptions() {
  const actions = Array.from(new Set(applyFilters(state.events, { ignoreAction: true }).map((event) => event.action).filter(Boolean))).sort();
  const currentValue = state.selectedFilter;

  actionFilter.innerHTML = "";

  const allOption = document.createElement("option");
  allOption.value = "all";
  allOption.textContent = "Все";
  actionFilter.appendChild(allOption);

  actions.forEach((action) => {
    const option = document.createElement("option");
    option.value = action;
    option.textContent = action;
    actionFilter.appendChild(option);
  });

  if (currentValue === "all" || actions.includes(currentValue)) {
    actionFilter.value = currentValue;
    state.selectedFilter = currentValue;
    return;
  }

  actionFilter.value = "all";
  state.selectedFilter = "all";
}

function syncTypeFilterOptions() {
  const types = Array.from(new Set(applyFilters(state.events, { ignoreType: true }).map((event) => event.resource_type).filter(Boolean))).sort();
  const currentValue = state.selectedType;

  typeFilter.innerHTML = "";

  const allOption = document.createElement("option");
  allOption.value = "all";
  allOption.textContent = "Все";
  typeFilter.appendChild(allOption);

  types.forEach((type) => {
    const option = document.createElement("option");
    option.value = type;
    option.textContent = type;
    typeFilter.appendChild(option);
  });

  if (currentValue === "all" || types.includes(currentValue)) {
    typeFilter.value = currentValue;
    state.selectedType = currentValue;
    return;
  }

  typeFilter.value = "all";
  state.selectedType = "all";
}

function syncScheduleFilterOptions() {
  const schedules = Array.from(new Set(applyFilters(state.events, { ignoreSchedule: true }).map((event) => displayName(event)).filter(Boolean))).sort();
  if (!scheduleFilter) {
    return;
  }

  if (scheduleFilterIsInput && scheduleOptions) {
    scheduleOptions.innerHTML = "";

    schedules.forEach((schedule) => {
      const option = document.createElement("option");
      option.value = schedule;
      scheduleOptions.appendChild(option);
    });

    scheduleFilter.placeholder = schedules.length > 0 ? "Все" : "Нет вариантов";
    return;
  }

  const currentValue = normalizeScheduleFilterValue(scheduleFilter.value);
  scheduleFilter.innerHTML = "";

  const allOption = document.createElement("option");
  allOption.value = "all";
  allOption.textContent = "Все";
  scheduleFilter.appendChild(allOption);

  schedules.forEach((schedule) => {
    const option = document.createElement("option");
    option.value = schedule;
    option.textContent = schedule;
    scheduleFilter.appendChild(option);
  });

  if (currentValue === "" || schedules.includes(currentValue)) {
    scheduleFilter.value = currentValue === "" ? "all" : currentValue;
    state.selectedSchedule = currentValue;
    return;
  }

  scheduleFilter.value = "all";
  state.selectedSchedule = "";
}

function applyFilters(events, options = {}) {
  const selectedAction = options.ignoreAction ? "all" : state.selectedFilter;
  const selectedType = options.ignoreType ? "all" : state.selectedType;
  const selectedSchedule = options.ignoreSchedule ? "" : state.selectedSchedule;
  const normalizedSchedule = normalizeScheduleFilterValue(selectedSchedule).toLowerCase();

  return events.filter((event) => {
    const matchesAction = selectedAction === "all" || event.action === selectedAction;
    const matchesType = selectedType === "all" || event.resource_type === selectedType;
    const matchesSchedule = normalizedSchedule === "" || displayName(event).toLowerCase().includes(normalizedSchedule);
    return matchesAction && matchesType && matchesSchedule;
  });
}

function groupEventsByDate(events) {
  return events.reduce((acc, event) => {
    acc[event.local_date] = acc[event.local_date] || [];
    acc[event.local_date].push(event);
    return acc;
  }, {});
}

function groupEventsByTime(events) {
  const groups = new Map();

  events.forEach((event) => {
    const key = event.local_time;
    if (!groups.has(key)) {
      groups.set(key, { time: key, events: [] });
    }
    groups.get(key).events.push(event);
  });

  return Array.from(groups.values()).sort((left, right) => left.time.localeCompare(right.time));
}

function groupEventsByHour(events, hour) {
  return groupEventsByTime(events).filter((group) => Number(group.time.slice(0, 2)) === hour);
}

function buildWeekDays(weekStart) {
  const weekdays = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];
  return Array.from({ length: 7 }, (_, index) => {
    const date = addDays(weekStart, index);
    return {
      date: formatDateLocal(date),
      weekday: weekdays[index],
      dayMonth: new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit" }).format(date),
    };
  });
}

function normalizeSelectedDate(events, fallbackDate, endDate) {
  if (state.selectedDate && state.selectedDate >= fallbackDate && state.selectedDate <= endDate) {
    return state.selectedDate;
  }
  const today = formatDateLocal(todayInTimezone());
  if (today >= fallbackDate && today <= endDate) {
    return today;
  }
  if (events.length > 0) {
    return events[0].local_date;
  }
  return fallbackDate;
}

function selectedDetailsTitle() {
  if (!state.selectedDate) {
    return "Выберите день";
  }
  if (state.selectedHour === null) {
    return formatHumanDate(state.selectedDate);
  }
  if (state.selectedTime !== null) {
    return `${formatHumanDate(state.selectedDate)}, ${shortTime(state.selectedTime)}`;
  }
  return `${formatHumanDate(state.selectedDate)}, ${String(state.selectedHour).padStart(2, "0")}:00`;
}

function emptyDetailsText() {
  if (!state.selectedDate) {
    return "Выберите день в календаре.";
  }
  if (state.selectedTime !== null) {
    return "В выбранное время событий нет.";
  }
  if (state.selectedHour !== null) {
    return "В выбранное время событий нет.";
  }
  return "На этот день событий нет.";
}

function startOfWeek(date) {
  const copy = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const offset = (copy.getDay() + 6) % 7;
  copy.setDate(copy.getDate() - offset);
  return copy;
}

function addDays(date, days) {
  const copy = new Date(date);
  copy.setDate(copy.getDate() + days);
  return copy;
}

function formatDateLocal(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function todayInTimezone() {
  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: state.timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
  const parts = formatter.formatToParts(new Date());
  const year = Number(parts.find((part) => part.type === "year").value);
  const month = Number(parts.find((part) => part.type === "month").value);
  const day = Number(parts.find((part) => part.type === "day").value);
  return new Date(year, month - 1, day);
}

function formatWeekTitle(weekStart) {
  const weekEnd = addDays(weekStart, 6);
  const formatter = new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "long" });
  return `${formatter.format(weekStart)} - ${formatter.format(weekEnd)}`;
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

function formatEventTooltip(event) {
  return [
    displayName(event),
    `${shortTime(event.local_time)} · ${event.action}`,
    `${event.resource_type} · ${event.resource_id}`,
    event.schedule_name,
    `Статус: ${formatStateLabel(event)}`,
  ].filter(Boolean).join("\n");
}

function displayName(event) {
  return event.schedule_display_name || event.schedule_name || "unnamed";
}

function eventColor(event) {
  const palette = [
    "#3ecf8e",
    "#ff8a65",
    "#59c3c3",
    "#f2c94c",
    "#8ab4f8",
    "#d386f7",
    "#f78fb3",
    "#a3e635",
    "#f59e0b",
    "#22d3ee",
  ];
  const key = displayName(event);
  let hash = 0;
  for (let i = 0; i < key.length; i++) {
    hash = (hash * 31 + key.charCodeAt(i)) >>> 0;
  }
  return palette[hash % palette.length];
}

function formatGroupTooltip(group) {
  if (group.events.length === 1) {
    return formatEventTooltip(group.events[0]);
  }

  return [
    `${shortTime(group.time)} · ${group.events.length} задач`,
    ...group.events.map((event) => `- ${displayName(event)} · ${event.action} · ${event.resource_type}`),
  ].join("\n");
}

function formatHourTooltip(groups) {
  return groups.map(formatGroupTooltip).join("\n\n");
}

function groupActionClass(group) {
  if (group.events.length === 0) {
    return "mixed";
  }

  const action = group.events[0].action;
  return group.events.every((event) => event.action === action) ? action : "mixed";
}

function normalizeScheduleFilterValue(value) {
  if (!value || value === "all") {
    return "";
  }
  return value.trim();
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
