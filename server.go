package main

import (
    "encoding/json"
    "fmt"
    "html/template"
    "io"
    "io/ioutil"
    "net/http"
    "strconv"
    "strings"

    _ "github.com/go-sql-driver/mysql"
    "github.com/labstack/echo/v4"
    "xorm.io/xorm"

    "github.com/readper/naming-server/models"
)
type Template struct {
    templates *template.Template
}
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
    return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
    engine, err := xorm.NewEngine("mysql", "root:password@tcp(localhost:3306)/")
    if err != nil {
        panic(err)
    }
    if err := engine.Ping();err != nil {
        panic(err)
    }
    // create data from json when db not exists
    fmt.Println("Checking Databases")
    if _,err := engine.Exec("CREATE DATABASE naming;"); err==nil {
        if _, err := engine.Exec("use naming;"); err != nil {
            panic(err)
        }

        if err := engine.Sync2(new(models.Word)); err != nil {
            panic(err)
        }
        if err := engine.Sync2(new(models.UnwantWord)); err != nil {
            panic(err)
        }
        fmt.Println("prepare data")
        var dicts []models.Dict

        data, err := ioutil.ReadFile("data/dict-revised.json")
        if err != nil {
            panic(err)
        }
        if err := json.Unmarshal(data, &dicts); err != nil {
            panic(err)
        }
        count:=0
        for _, dict := range dicts {
            count++
            if count%1000 == 0{
                fmt.Printf("%8d / %8d\n",count,len(dicts))
            }
            if dict.StrokeCount == 0 {
                continue
            }
            if strings.Contains(dict.Title, "{[") {
                continue
            }
            bopomofo := ""
            meaning := ""
            for _, h := range *dict.Heteronyms {
                if bopomofo == ""{
                    bopomofo=h.Bopomofo
                } else {
                    bopomofo = fmt.Sprintf("%s\n%s", bopomofo, h.Bopomofo)
                }
                for _, d := range *h.Definitions {
                    if meaning==""{
                        meaning = fmt.Sprintf("%s %s", d.Def, d.Link)
                    }else {
                        meaning = fmt.Sprintf("%s\n%s %s", meaning, d.Def, d.Link)
                    }
                }
            }
            word := models.Word{
                Word:     dict.Title,
                Bopomofo: bopomofo,
                Meaning:  meaning,
                StrokeCount: dict.StrokeCount,
            }
            if _, err := engine.Insert(word); err != nil {
                panic(err)
            }
        }
    }
    if _, err := engine.Exec("use naming;"); err != nil {
        panic(err)
    }

    e := echo.New()
    t := &Template{
        templates: template.Must(template.ParseGlob("views/*.html")),
    }
    e.Renderer = t
    e.GET("/", func(c echo.Context) error {
        return c.String(http.StatusOK, "Hello, World!")
    })
    e.GET("/words", func(c echo.Context) error {
        strokeCount := c.QueryParam("StrokeCount")
        words := make([]models.Word, 0)
        if err := engine.Where("stroke_count = ?",strokeCount).Find(&words);err!=nil{
            return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
        }
        unwants := make([]models.UnwantWord, 0)
        if err := engine.Where("naming_id = 1 and stroke_count = ?", strokeCount).Find(&unwants);err != nil {
            return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
        }
        unwantMap := map[int64]struct{}{}
        for _, unwant := range unwants{
            unwantMap[unwant.WordId]=struct{}{}
        }
        for i := range words{
            if _, ok := unwantMap[words[i].Id]; ok {
                words[i].Unwant = true
            }
        }
        err = c.Render(http.StatusOK, "words.html", map[string]interface{}{"Words":words})
        if err != nil {
            fmt.Println(err)
        }
        return err
    })
    e.PATCH("/unwant_words/:id", func(c echo.Context) error {
        wordID, _ := strconv.ParseInt(c.Param("id"),10,64)
        // try find unwant_word
        unwants := make([]models.UnwantWord, 0)
        if err := engine.Where("word_id = ? and naming_id = 1",wordID).Find(&unwants);err != nil {
            return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
        }
        if len(unwants) == 0 {
            words := make([]models.Word, 0)
            if err := engine.Where("id = ?",wordID).Find(&words);err != nil {
                return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
            }
            unwant:= models.UnwantWord{
                NamingId:    1,
                WordId:      wordID,
                StrokeCount: words[0].StrokeCount,
            }
            if _,err := engine.Insert(unwant);err!=nil{
                return c.String(http.StatusInternalServerError, "insert failed!\n"+err.Error())
            }
            return c.String(http.StatusOK, "created")
        }
        unwant:= models.UnwantWord{}
        if _,err := engine.Where("word_id = ? and naming_id = 1",wordID).Delete(&unwant);err != nil {
            return c.String(http.StatusInternalServerError, "Delete failed!\n"+err.Error())
        }
        return c.String(http.StatusOK, "removed")
    })
    e.Logger.Fatal(e.Start(":1323"))
}