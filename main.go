package main
import (
  "fmt"
  "net/http"
  "html/template"

  "database/sql"

	_ "github.com/mattn/go-sqlite3"
  "github.com/urfave/negroni"
  "encoding/json"
  "net/url"
  "io/ioutil"
  "encoding/xml"

)
type SearchResult struct{
  Title string `xml:"title,attr"`
  Author string `xml:"author,attr"`
  Year string `xml:"hyr,attr"`
  ID string `xml:"owi,attr"`

}
type ClassifyBookResponse struct{
  BookData struct{
    Title string `xml:"title,atrr"`
    Author string `xml:"author,attr"`
    ID string `xml:"owi,attr"`
  } `xml:"work"`
  Classification struct{
    MostPopular string `xml:"sfa,attr"`
  } `xml:"recommendations>ddc>mostPopular"`
}
type Page struct{
  Name string
  DBSTATUS bool
}
type ClassifySearchResponse struct{
  Results []SearchResult `xml:"works>work"`
}
var db *sql.DB
func verifyDatabase(w http.ResponseWriter, r *http.Request, next http.HandlerFunc){
  if err := db.Ping(); err != nil {
    http.Error(w,err.Error(), http.StatusInternalServerError)
    return
  }
  next(w,r)
}
func main(){
  templates := template.Must(template.ParseFiles("templates/index.html"))
  db, _ = sql.Open("sqlite3","dev.db")

  mux := http.NewServeMux()

  mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
    p := Page{Name: "Gopher"}
    if name := r.FormValue("name"); name != ""{
       p.Name = name
    }
    p.DBSTATUS = db.Ping() == nil
    if err := templates.ExecuteTemplate(w,"index.html",p); err != nil{
      http.Error(w,err.Error(),http.StatusInternalServerError)
    }
  })
  mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request){
    var results []SearchResult
    var err error
    if results, err = search(r.FormValue("search")); err != nil{
      http.Error(w,err.Error(), http.StatusInternalServerError)
    }


    encoder := json.NewEncoder(w)
    if err := encoder.Encode(results); err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
    }

  })
  mux.HandleFunc("/books/add",func(w http.ResponseWriter, r *http.Request){
    var book ClassifyBookResponse
    var err error
    if book, err = find(r.FormValue("id")); err != nil{
      fmt.Println("error getting id")
      http.Error(w,err.Error(), http.StatusInternalServerError)
    }
    if err = db.Ping(); err != nil{
      fmt.Println("database not up")
      http.Error(w,err.Error(), http.StatusInternalServerError)
    }
    _, err = db.Exec("insert into books (pk, title, author, id, classification) values (?,?,?,?,?)",
      nil,book.BookData.Title,book.BookData.Author,book.BookData.ID,book.Classification.MostPopular)

  })
  n := negroni.Classic()
  n.Use(negroni.HandlerFunc(verifyDatabase))
  n.UseHandler(mux)

  n.Run(":8080")
}
func search(query string) ([]SearchResult, error){

  var c ClassifySearchResponse
  body, err := classifyAPI("http://classify.oclc.org/classify2/Classify?&summary=true&title=" + url.QueryEscape(query))
  if err != nil {
    return []SearchResult{},err
  }
  err = xml.Unmarshal(body, &c)
  return c.Results, err
}
func find(id string) (ClassifyBookResponse, error){
  var c ClassifyBookResponse
  body, err := classifyAPI("http://classify.oclc.org/classify2/Classify?&summary=true&owi=" + url.QueryEscape(id))
  if err != nil {
    return ClassifyBookResponse{}, err
  }
  err = xml.Unmarshal(body, &c)
  return c, err
}
func classifyAPI(url string) ([]byte, error){
  var resp *http.Response
  var err error
  if resp, err = http.Get(url); err != nil{
    return []byte{},err
  }
  defer resp.Body.Close()

  return ioutil.ReadAll(resp.Body)
}
