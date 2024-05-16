package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "github.com/rs/cors"
    "github.com/swaggo/http-swagger" 
    _ "PatientRecordsBackend/docs"                    // Import the generated docs
    _ "github.com/mattn/go-sqlite3"  // Import SQLite driver
)

// @title Medical Records API
// @version 1.0
// @description This is a sample server for managing medical records.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /api

// Patient represents the details of a patient.
type Patient struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}

// HealthRecord defines the interface for different types of health records
type HealthRecord interface {
    Type() string
    Details() interface{}
}

// VisitRecord represents a visit health record
type VisitRecord struct {
    Date       string `json:"date"`
    Reason     string `json:"reason"`
    DoctorName string `json:"doctorName"`
    Hospital   string `json:"hospital"`
}

// Type returns the type of the visit record
func (v VisitRecord) Type() string {
    return "visit"
}

// Details returns the details of the visit record
func (v VisitRecord) Details() interface{} {
    return v
}

// Record represents a patient's records including visit, treatment, and diagnostic records
type Record struct {
    Patient        Patient            `json:"patient"`
    VisitRecords   []VisitRecord      `json:"visitRecords"`
    TreatmentRecs  []TreatmentRecord  `json:"treatmentRecords"`
    DiagRecords    []DiagnosticRecord `json:"diagnosticRecords"`
}

// TreatmentRecord represents a treatment health record
type TreatmentRecord struct {
    Date      string `json:"date"`
    Treatment string `json:"treatment"`
    Outcome   string `json:"outcome"`
    Hospital  string `json:"hospital"`
}

// Type returns the type of the treatment record
func (t TreatmentRecord) Type() string {
    return "treatment"
}

// Details returns the details of the treatment record
func (t TreatmentRecord) Details() interface{} {
    return t
}

// DiagnosticRecord represents a diagnostic health record
type DiagnosticRecord struct {
    Date       string `json:"date"`
    Diagnosis  string `json:"diagnosis"`
    Specialist string `json:"specialist"`
    Hospital   string `json:"hospital"`
}

// Type returns the type of the diagnostic record
func (d DiagnosticRecord) Type() string {
    return "diagnostic"
}

// Details returns the details of the diagnostic record
func (d DiagnosticRecord) Details() interface{} {
    return d
}

var db *sql.DB



func main() {
    var err error
    db, err = sql.Open("sqlite3", "file:medical_records.db?cache=shared&mode=rwc")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    mux := http.NewServeMux()
    const apiBasePath = "/api"

    mux.HandleFunc(apiBasePath+"/patients/", patientsHandler)
    mux.HandleFunc(apiBasePath+"/records", recordsHandler)
    mux.HandleFunc(apiBasePath+"/addRecord", addRecordHandler)
    mux.HandleFunc(apiBasePath+"/search", searchHandler)

    // Create a new CORS handler with default options
    corsHandler := cors.Default().Handler(mux)

    // Setup Swagger documentation endpoint
    mux.Handle("/swagger/", httpSwagger.WrapHandler)

    fmt.Println("Server is running on http://localhost:8080/api")
    log.Fatal(http.ListenAndServe(":8080", corsHandler))
}


// @Summary Get patient by ID
// @Description Get a patient's details by ID.
// @ID get-patient-by-id
// @Accept  json
// @Produce  json
// @Param   id   path    string     true  "Patient ID"
// @Success 200 {object} Patient
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /patients/{id} [get]
func patientsHandler(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimPrefix(r.URL.Path, "/api/patients/")

    switch r.Method {
    case "GET":
        var patient Patient
        err := db.QueryRow("SELECT id, name, age FROM Patients WHERE id = ?", id).Scan(&patient.ID, &patient.Name, &patient.Age)
        if err == sql.ErrNoRows {
            http.NotFound(w, r)
            return
        } else if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        json.NewEncoder(w).Encode(patient)

    case "PUT":
        // Handle PUT request to update patient information
        var patient Patient
        if err := json.NewDecoder(r.Body).Decode(&patient); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        _, err := db.Exec("UPDATE Patients SET name = ?, age = ? WHERE id = ?", patient.Name, patient.Age, id)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        fmt.Fprintf(w, "Patient information updated successfully")
    default:
        http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
    }
}

func insertVisits(tx *sql.Tx, record *Record) error {
    for _, visit := range record.VisitRecords {
        _, err := tx.Exec("INSERT INTO Visits (patient_id, date, reason, doctor_name, hospital) VALUES (?, ?, ?, ?, ?)",
            record.Patient.ID, visit.Date, visit.Reason, visit.DoctorName, visit.Hospital)
        if err != nil {
            return err
        }
    }
    return nil
}

func insertTreatments(tx *sql.Tx, record *Record) error {
    for _, treatment := range record.TreatmentRecs {
        _, err := tx.Exec("INSERT INTO Treatments (patient_id, date, treatment, outcome, hospital) VALUES (?, ?, ?, ?, ?)",
            record.Patient.ID, treatment.Date, treatment.Treatment, treatment.Outcome, treatment.Hospital)
        if err != nil {
            return err
        }
    }
    return nil
}

func insertDiagnostics(tx *sql.Tx, record *Record) error {
    for _, diagnostic := range record.DiagRecords {
        _, err := tx.Exec("INSERT INTO Diagnostics (patient_id, date, diagnosis, specialist, hospital) VALUES (?, ?, ?, ?, ?)",
            record.Patient.ID, diagnostic.Date, diagnostic.Diagnosis, diagnostic.Specialist, diagnostic.Hospital)
        if err != nil {
            return err
        }
    }
    return nil
}


// @Summary Search patient records
// @Description Search patient records by patient ID or hospital.
// @ID search-patient-records
// @Accept  json
// @Produce  json
// @Param   patientId   query    string     false  "Patient ID"
// @Param   hospital    query    string     false  "Hospital"
// @Success 200 {array} Record
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /search [get]
func searchHandler(w http.ResponseWriter, r *http.Request) {
    patientID := r.URL.Query().Get("patientId")
    hospital := r.URL.Query().Get("hospital")

    // Validate input parameters
    if patientID == "" && hospital == "" {
        http.Error(w, "Please provide either a patientId or hospital parameter", http.StatusBadRequest)
        return
    }

    // Prepare SQL to fetch all related records
    baseQuery := `
        SELECT 
            p.id, p.name, p.age, 
            v.date AS visit_date, v.reason, v.doctor_name, v.hospital AS visit_hospital,
            t.date AS treatment_date, t.treatment, t.outcome, t.hospital AS treatment_hospital,
            d.date AS diagnostic_date, d.diagnosis, d.specialist, d.hospital AS diagnostic_hospital
        FROM Patients p
        LEFT JOIN Visits v ON p.id = v.patient_id
        LEFT JOIN Treatments t ON p.id = t.patient_id
        LEFT JOIN Diagnostics d ON p.id = d.patient_id
    `
    var queryParams []interface{}
    var conditions []string

    if patientID != "" {
        conditions = append(conditions, "p.id = ?")
        queryParams = append(queryParams, patientID)
    }
    if hospital != "" {
        hospitalCondition := "(v.hospital = ? OR t.hospital = ? OR d.hospital = ?)"
        conditions = append(conditions, hospitalCondition, hospitalCondition, hospitalCondition)
        queryParams = append(queryParams, hospital, hospital, hospital)
    }

    // Build the final query with conditions
    if len(conditions) > 0 {
        baseQuery += " WHERE " + strings.Join(conditions, " AND ")
    }

    // Execute the query
    rows, err := db.Query(baseQuery, queryParams...)
    if err != nil {
        http.Error(w, "Database query error: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var records []Record
    recordMap := make(map[string]*Record)

    // Iterate over the result rows
    for rows.Next() {
        var (
            id, name, visitHospital, treatmentHospital, diagnosticHospital string
            age                                                              int
            vDate, vReason, vDoctorName                                      string
            tDate, tTreatment, tOutcome                                      string
            dDate, dDiagnosis, dSpecialist                                   string
        )

        err = rows.Scan(
            &id, &name, &age,
            &vDate, &vReason, &vDoctorName, &visitHospital,
            &tDate, &tTreatment, &tOutcome, &treatmentHospital,
            &dDate, &dDiagnosis, &dSpecialist, &diagnosticHospital,
        )
        if err != nil {
            http.Error(w, "Failed to parse database results: "+err.Error(), http.StatusInternalServerError)
            return
        }

        // Initialize record if it doesn't exist
        if _, exists := recordMap[id]; !exists {
            recordMap[id] = &Record{
                Patient: Patient{
                    ID:   id,
                    Name: name,
                    Age:  age,
                },
                VisitRecords:   []VisitRecord{},
                TreatmentRecs:  []TreatmentRecord{},
                DiagRecords:    []DiagnosticRecord{},
            }
        }

        // Append visit record if data exists
        if vDate != "" {
            recordMap[id].VisitRecords = append(recordMap[id].VisitRecords, VisitRecord{
                Date:       vDate,
                Reason:     vReason,
                DoctorName: vDoctorName,
                Hospital:   visitHospital,
            })
        }

        // Append treatment record if data exists
        if tDate != "" {
            recordMap[id].TreatmentRecs = append(recordMap[id].TreatmentRecs, TreatmentRecord{
                Date:      tDate,
                Treatment: tTreatment,
                Outcome:   tOutcome,
                Hospital:  treatmentHospital,
            })
        }

        // Append diagnostic record if data exists
        if dDate != "" {
            recordMap[id].DiagRecords = append(recordMap[id].DiagRecords, DiagnosticRecord{
                Date:       dDate,
                Diagnosis:  dDiagnosis,
                Specialist: dSpecialist,
                Hospital:   diagnosticHospital,
            })
        }
    }

    // Convert map to slice
    for _, rec := range recordMap {
        records = append(records, *rec)
    }

    // Check for empty result set
    if len(records) == 0 {
        http.NotFound(w, r)
        return
    }

    // Send the records as JSON
    json.NewEncoder(w).Encode(records)
}


// Updates an existing visit record based on provided data
func updateVisitRecord(tx *sql.Tx, patientID string, data map[string]interface{}) error {
    date := data["date"].(string)
    reason := data["reason"].(string)
    doctorName := data["doctorName"].(string)
    hospital := data["hospital"].(string)

    _, err := tx.Exec("UPDATE Visits SET date = ?, reason = ?, doctor_name = ?, hospital = ? WHERE patient_id = ? AND date = ?",
        date, reason, doctorName, hospital, patientID, date)
    return err
}

// Updates an existing treatment record based on provided data
func updateTreatmentRecord(tx *sql.Tx, patientID string, data map[string]interface{}) error {
    date := data["date"].(string)
    treatment := data["treatment"].(string)
    outcome := data["outcome"].(string)
    hospital := data["hospital"].(string)

    _, err := tx.Exec("UPDATE Treatments SET date = ?, treatment = ?, outcome = ?, hospital = ? WHERE patient_id = ? AND date = ?",
        date, treatment, outcome, hospital, patientID, date)
    return err
}

// Updates an existing diagnostic record based on provided data
func updateDiagnosticRecord(tx *sql.Tx, patientID string, data map[string]interface{}) error {
    date := data["date"].(string)
    diagnosis := data["diagnosis"].(string)
    specialist := data["specialist"].(string)
    hospital := data["hospital"].(string)

    _, err := tx.Exec("UPDATE Diagnostics SET date = ?, diagnosis = ?, specialist = ?, hospital = ? WHERE patient_id = ? AND date = ?",
        date, diagnosis, specialist, hospital, patientID, date)
    return err
}

// @Summary Add a new record
// @Description Add a new record for a patient.
// @ID add-record
// @Accept  json
// @Produce  json
// @Param   record  body    Record     true  "Record"
// @Success 201 {object} Record
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /records [post]
func recordsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
        return
    }

    var record Record
    if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    tx, err := db.Begin()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if _, err = tx.Exec("INSERT INTO Patients (id, name, age) VALUES (?, ?, ?)", record.Patient.ID, record.Patient.Name, record.Patient.Age); err != nil {
        tx.Rollback()
        http.Error(w, "Failed to insert new patient: "+err.Error(), http.StatusInternalServerError)
        return
    }

    if err := insertVisits(tx, &record); err != nil {
        tx.Rollback()
        http.Error(w, "Failed to insert visits: "+err.Error(), http.StatusInternalServerError)
        return
    }
    if err := insertTreatments(tx, &record); err != nil {
        tx.Rollback()
        http.Error(w, "Failed to insert treatments: "+err.Error(), http.StatusInternalServerError)
        return
    }
    if err := insertDiagnostics(tx, &record); err != nil {
        tx.Rollback()
        http.Error(w, "Failed to insert diagnostics: "+err.Error(), http.StatusInternalServerError)
        return
    }

    if err = tx.Commit(); err != nil {
        http.Error(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(record)
}



func addRecordHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        httpRespondWithError(w, http.StatusMethodNotAllowed, "Unsupported method")
        return
    }

    var genericMap map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&genericMap); err != nil {
        httpRespondWithError(w, http.StatusBadRequest, err.Error())
        return
    }

    patientID, recordType, recordData := genericMap["patientId"].(string), genericMap["type"].(string), genericMap["record"].(map[string]interface{})
    tx, err := db.Begin()
    if err != nil {
        httpRespondWithError(w, http.StatusInternalServerError, "Failed to start transaction: "+err.Error())
        return
    }

    if err := updateRecord(tx, patientID, recordType, recordData); err != nil {
        tx.Rollback()
        httpRespondWithError(w, http.StatusInternalServerError, err.Error())
        return
    }

    if err := tx.Commit(); err != nil {
        httpRespondWithError(w, http.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
        return
    }

    fmt.Fprintln(w, "Record updated successfully")
}

func updateRecord(tx *sql.Tx, patientID, recordType string, data map[string]interface{}) error {
    switch recordType {
    case "visit":
        return updateVisitRecord(tx, patientID, data)
    case "treatment":
        return updateTreatmentRecord(tx, patientID, data)
    case "diagnostic":
        return updateDiagnosticRecord(tx, patientID, data)
    default:
        return fmt.Errorf("Invalid record type")
    }
}

// Utility function to send error responses
func httpRespondWithError(w http.ResponseWriter, statusCode int, msg string) {
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
