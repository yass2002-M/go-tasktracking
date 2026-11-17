package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"todo_web/models"
)

const dataFile = "data/tasks.json"

// Ce fichier conserve le chronomètre actif et garantit un seul chrono à la fois.
const activeTimerFile = "data/active_timer.json"

type ActiveTimer struct {
	TaskID    int       `json:"task_id"`
	StartedAt time.Time `json:"started_at"`
}

type indexPageData struct {
	Today                []models.Task
	Late                 []models.Task
	General              []models.Task
	ActiveTimer          *ActiveTimer
	ActiveTimerTaskTitle string
}

var timerMu sync.Mutex

var funcMap = template.FuncMap{
	"div": func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"mod": func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a % b
	},
	"now": func() time.Time {
		return time.Now()
	},
	"formatDuration": func(minutes, seconds int) string {
		totalMinutes := minutes + seconds/60
		if totalMinutes < 60 {
			return fmt.Sprintf("%d min", totalMinutes)
		}
		return fmt.Sprintf("%d h %d min", totalMinutes/60, totalMinutes%60)
	},
}

var tmpl = template.Must(template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html"))

func main() {
	// Les chemins des templates et des données sont relatifs au dossier de l'exécutable.
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Erreur: impossible de trouver l'exécutable", err)
	}
	workDir := filepath.Dir(exePath)
	err = os.Chdir(workDir)
	if err != nil {
		log.Fatal("Erreur: impossible de changer de répertoire", err)
	}
	http.HandleFunc("/note", noteHandler)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/update", updateHandler)
	http.HandleFunc("/timer/start", timerStartHandler)
	http.HandleFunc("/timer/stop", timerStopHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/set-today", setTodayHandler)

	http.HandleFunc("/calendar-view", calendarViewController)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Serveur démarré sur http://localhost:8090")
	go func() {
		time.Sleep(time.Second)
		if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://localhost:8090").Start(); err != nil {
			log.Println("Impossible d'ouvrir le navigateur automatiquement:", err)
		}
	}()
	fmt.Println("Laissez cette fenêtre ouverte pour garder l'application active. Ctrl+C pour arrêter.")
	err = http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Println("Erreur serveur:", err)
	}
}

func loadTasks() []models.Task {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return []models.Task{}
	}
	var tasks []models.Task
	_ = json.Unmarshal(data, &tasks)
	return tasks
}

func saveTasks(tasks []models.Task) {
	data, _ := json.MarshalIndent(tasks, "", "  ")
	os.MkdirAll("data", 0755)
	_ = os.WriteFile(dataFile, data, 0644)
}

func loadActiveTimer() *ActiveTimer {
	data, err := os.ReadFile(activeTimerFile)
	if err != nil {
		return nil
	}
	var active ActiveTimer
	if json.Unmarshal(data, &active) != nil || active.TaskID == 0 || active.StartedAt.IsZero() {
		return nil
	}
	return &active
}

func saveActiveTimer(active *ActiveTimer) {
	if active == nil {
		_ = os.Remove(activeTimerFile)
		return
	}
	data, err := json.MarshalIndent(active, "", "  ")
	if err == nil {
		_ = os.WriteFile(activeTimerFile, data, 0644)
	}
}

func writeTimerResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func timerStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeTimerResponse(w, http.StatusMethodNotAllowed, "Méthode non autorisée.")
		return
	}
	id, _ := strconv.Atoi(r.FormValue("id"))
	timerMu.Lock()
	defer timerMu.Unlock()
	tasks := loadTasks()
	var taskTitle string
	for _, task := range tasks {
		if task.ID == id {
			taskTitle = task.Title
			break
		}
	}
	if taskTitle == "" {
		writeTimerResponse(w, http.StatusNotFound, "Cette tâche est introuvable.")
		return
	}
	if active := loadActiveTimer(); active != nil {
		for _, task := range tasks {
			if task.ID == active.TaskID {
				writeTimerResponse(w, http.StatusConflict, "Veuillez arrêter d'abord le chrono sur « "+task.Title+" ».")
				return
			}
		}
		// Libère un chrono orphelin si sa tâche a été supprimée.
		saveActiveTimer(nil)
	}
	saveActiveTimer(&ActiveTimer{TaskID: id, StartedAt: time.Now()})
	writeTimerResponse(w, http.StatusOK, "Chronomètre démarré.")
}

func timerStopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeTimerResponse(w, http.StatusMethodNotAllowed, "Méthode non autorisée.")
		return
	}
	timerMu.Lock()
	defer timerMu.Unlock()
	active := loadActiveTimer()
	if active == nil {
		writeTimerResponse(w, http.StatusConflict, "Aucun chronomètre n'est en cours.")
		return
	}
	tasks := loadTasks()
	elapsed := int(time.Since(active.StartedAt).Seconds())
	if elapsed < 0 {
		elapsed = 0
	}
	for i := range tasks {
		if tasks[i].ID == active.TaskID {
			totalSeconds := tasks[i].WorkMinutes*60 + tasks[i].WorkSeconds + elapsed
			tasks[i].WorkMinutes = totalSeconds / 60
			tasks[i].WorkSeconds = totalSeconds % 60
			break
		}
	}
	saveTasks(tasks)
	saveActiveTimer(nil)
	writeTimerResponse(w, http.StatusOK, "Temps ajouté à la tâche.")
}

// Passe automatiquement en retard les tâches dont l'échéance est dépassée.
func updateStatuses(tasks []models.Task) []models.Task {
	today := time.Now()
	for i := range tasks {
		if (tasks[i].Status == "today" || tasks[i].Status == "general") && tasks[i].Deadline.Before(today) {
			tasks[i].Status = "late"
		}
	}
	return tasks
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tasks := loadTasks()
	tasks = updateStatuses(tasks)
	saveTasks(tasks)

	var todayTasks, lateTasks, generalTasks []models.Task
	for _, t := range tasks {
		switch t.Status {
		case "today":
			todayTasks = append(todayTasks, t)
		case "late":
			lateTasks = append(lateTasks, t)
		default:
			generalTasks = append(generalTasks, t)
		}
	}

	sort.Slice(todayTasks, func(i, j int) bool {
		if todayTasks[i].Importance != todayTasks[j].Importance {
			return todayTasks[i].Importance > todayTasks[j].Importance
		}
		return todayTasks[i].Deadline.Before(todayTasks[j].Deadline)
	})

	sort.Slice(lateTasks, func(i, j int) bool {
		return lateTasks[i].Deadline.Before(lateTasks[j].Deadline)
	})

	sort.Slice(generalTasks, func(i, j int) bool {
		if generalTasks[i].Deadline.Equal(generalTasks[j].Deadline) {
			return generalTasks[i].Importance > generalTasks[j].Importance
		}
		return generalTasks[i].Deadline.Before(generalTasks[j].Deadline)
	})

	active := loadActiveTimer()
	activeTitle := ""
	if active != nil {
		for _, task := range tasks {
			if task.ID == active.TaskID {
				activeTitle = task.Title
				break
			}
		}
	}
	data := indexPageData{
		Today:                todayTasks,
		Late:                 lateTasks,
		General:              generalTasks,
		ActiveTimer:          active,
		ActiveTimerTaskTitle: activeTitle,
	}

	_ = tmpl.Execute(w, data)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.ParseForm()
	title := r.FormValue("title")
	deadlineStr := r.FormValue("deadline")
	impStr := r.FormValue("importance")
	deadline, _ := time.Parse("2006-01-02", deadlineStr)
	importance, _ := strconv.Atoi(impStr)

	tasks := loadTasks()
	maxID := 0
	for _, t := range tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}

	tasks = append(tasks, models.Task{
		ID:          maxID + 1,
		Title:       title,
		Deadline:    deadline,
		Importance:  importance,
		WorkMinutes: 0,
		Status:      "general",
	})

	saveTasks(tasks)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	action := r.FormValue("action")

	tasks := loadTasks()
	for i := range tasks {
		if tasks[i].ID == id {
			if action == "+" {
				tasks[i].WorkMinutes += 20
			} else if action == "-" && tasks[i].WorkMinutes >= 20 {
				tasks[i].WorkMinutes -= 20
			}
			break
		}
	}
	saveTasks(tasks)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	timerMu.Lock()
	if active := loadActiveTimer(); active != nil && active.TaskID == id {
		saveActiveTimer(nil)
	}
	tasks := loadTasks()
	newTasks := []models.Task{}
	for _, t := range tasks {
		if t.ID != id {
			newTasks = append(newTasks, t)
		}
	}
	saveTasks(newTasks)
	timerMu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	title := r.FormValue("title")
	deadlineStr := r.FormValue("deadline")
	importance, _ := strconv.Atoi(r.FormValue("importance"))
	deadline, _ := time.Parse("2006-01-02", deadlineStr)

	tasks := loadTasks()
	for i := range tasks {
		if tasks[i].ID == id {
			tasks[i].Title = title
			tasks[i].Deadline = deadline
			tasks[i].Importance = importance
			break
		}
	}
	saveTasks(tasks)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func setTodayHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	tasks := loadTasks()
	for i := range tasks {
		if tasks[i].ID == id {
			tasks[i].Status = "today"
			break
		}
	}
	saveTasks(tasks)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func noteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.ParseForm()
	id, _ := strconv.Atoi(r.FormValue("id"))
	note := r.FormValue("note")

	tasks := loadTasks()
	for i := range tasks {
		if tasks[i].ID == id {
			tasks[i].Note = note
			break
		}
	}
	saveTasks(tasks)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func calendarViewController(w http.ResponseWriter, r *http.Request) {
	tasks := loadTasks()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if r.URL.Query().Get("year") != "" {
		year, _ = strconv.Atoi(r.URL.Query().Get("year"))
	}
	if r.URL.Query().Get("month") != "" {
		month, _ = strconv.Atoi(r.URL.Query().Get("month"))
	}

	tasksByDate := make(map[string][]models.Task)
	for _, task := range tasks {
		dateKey := task.Deadline.Format("2006-01-02")
		tasksByDate[dateKey] = append(tasksByDate[dateKey], task)
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	startOffset := int(firstDay.Weekday())

	emptyDays := make([]int, startOffset)
	for i := 0; i < startOffset; i++ {
		emptyDays[i] = i
	}

	type DayData struct {
		Day      int
		HasTasks bool
		Tasks    []models.Task
		IsToday  bool
	}

	var calendarDays []DayData
	today := time.Now()

	for d := 1; d <= lastDay.Day(); d++ {
		currentDate := time.Date(year, time.Month(month), d, 0, 0, 0, 0, time.UTC)
		dateKey := currentDate.Format("2006-01-02")
		tasksForDay := tasksByDate[dateKey]

		sort.Slice(tasksForDay, func(i, j int) bool {
			return tasksForDay[i].Importance > tasksForDay[j].Importance
		})

		calendarDays = append(calendarDays, DayData{
			Day:      d,
			HasTasks: len(tasksForDay) > 0,
			Tasks:    tasksForDay,
			IsToday:  currentDate.Format("2006-01-02") == today.Format("2006-01-02"),
		})
	}

	prevMonth := firstDay.AddDate(0, -1, 0)
	nextMonth := firstDay.AddDate(0, 1, 0)

	data := struct {
		Year      int
		Month     int
		MonthName string
		EmptyDays []int
		Days      []DayData
		PrevYear  int
		PrevMonth int
		NextYear  int
		NextMonth int
	}{
		Year:      year,
		Month:     month,
		MonthName: firstDay.Format("January"),
		EmptyDays: emptyDays,
		Days:      calendarDays,
		PrevYear:  prevMonth.Year(),
		PrevMonth: int(prevMonth.Month()),
		NextYear:  nextMonth.Year(),
		NextMonth: int(nextMonth.Month()),
	}

	tmplCal, err := template.ParseFiles("templates/calendar_view.html")
	if err != nil {
		log.Println("Erreur template:", err)
		http.Error(w, "Erreur chargement template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmplCal.Execute(w, data)
	if err != nil {
		log.Println("Erreur execution:", err)
		http.Error(w, "Erreur execution: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
