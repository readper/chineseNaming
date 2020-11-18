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
	"xorm.io/builder"
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
	//engine, err := xorm.NewEngine("mysql", "root:@tcp(localhost:3306)/?charset=utf8mb4")
	engine, err := xorm.NewEngine("mysql", "media17:media17@tcp(localhost:3306)/?charset=utf8mb4")
	if err != nil {
		panic(err)
	}
	if err := engine.Ping(); err != nil {
		panic(err)
	}
	// create data from json when db not exists
	fmt.Println("Checking Databases")
	if _, err := engine.Exec("CREATE DATABASE naming;"); err == nil {
		if _, err := engine.Exec("use naming;"); err != nil {
			panic(err)
		}

		if err := engine.Sync2(new(models.Word)); err != nil {
			panic(err)
		}
		if err := engine.Sync2(new(models.UnwantWord)); err != nil {
			panic(err)
		}
		if err := engine.Sync2(new(models.Order)); err != nil {
			panic(err)
		}
		if err := engine.Sync2(new(models.UnwantName)); err != nil {
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
		count := 0
		for _, dict := range dicts {
			count++
			if count%1000 == 0 {
				fmt.Printf("%8d / %8d\n", count, len(dicts))
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
				if bopomofo == "" {
					bopomofo = h.Bopomofo
				} else {
					bopomofo = fmt.Sprintf("%s\n%s", bopomofo, h.Bopomofo)
				}
				for _, d := range *h.Definitions {
					if meaning == "" {
						meaning = fmt.Sprintf("%s %s", d.Def, d.Link)
					} else {
						meaning = fmt.Sprintf("%s\n%s %s", meaning, d.Def, d.Link)
					}
				}
			}
			word := models.Word{
				Word:        dict.Title,
				Bopomofo:    bopomofo,
				Meaning:     meaning,
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
	//if err := engine.Sync2(new(models.UnwantName)); err != nil {
	//	panic(err)
	//}
	e := echo.New()
	t := &Template{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
	e.Renderer = t
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/names", func(c echo.Context) error {
		orders := make([]models.Order, 0)
		if _, err := engine.Exec("use naming;"); err != nil {
			panic(err)
		}
		if err := engine.Where("naming_id = 1").Find(&orders); err != nil {
			return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
		}
		orderWords := map[int64][]models.Word{}
		minOrder := int64(999)
		maxOrder := int64(0)
		for _, order := range orders {
			if order.Order > maxOrder {
				maxOrder = order.Order
			}
			if order.Order < minOrder {
				minOrder = order.Order
			}
			unwants := make([]models.UnwantWord, 0)
			if err := engine.Where("naming_id = 1").Find(&unwants); err != nil {
				return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
			}
			unwantsWordIDs := make([]int64, 0)
			for _, unwant := range unwants {
				unwantsWordIDs = append(unwantsWordIDs, unwant.WordId)
			}
			words := make([]models.Word, 0)
			if _, err := engine.Exec("use naming;"); err != nil {
				panic(err)
			}
			if err := engine.Where(builder.And(builder.Eq{"stroke_count": order.StrokeCount}, builder.NotIn("id", unwantsWordIDs))).Find(&words); err != nil {
				return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
			}
			for _, word := range words {
				orderWords[order.Order] = append(orderWords[order.Order], word)
			}
		}
		names := make([][]models.Word, 0)
		for _, word := range orderWords[minOrder] {
			names = append(names, []models.Word{word})
		}
		for i := minOrder + 1; i <= maxOrder; i++ {
			newNames := make([][]models.Word, 0)
			for _, name := range names {
				for _, word := range orderWords[i] {
					newName := append(name, word)
					newNames = append(newNames, newName)
				}
			}
			names = newNames
		}
		// change to map for check unwant
		mapNames := make(map[string][]models.Word,0)
		for _, name := range names{
			wordIDs := []string{}
			for _, word :=range name{
				wordIDs = append(wordIDs, fmt.Sprintf("%d",word.Id))
			}
			mapNames[strings.Join(wordIDs,"_")] = name
		}
		err = c.Render(http.StatusOK, "names.html", map[string]interface{}{"Names": mapNames})
		if err != nil {
			fmt.Println(err)
		}
		return err
	})
	e.GET("/words", func(c echo.Context) error {
		strokeCount := c.QueryParam("StrokeCount")
		words := make([]models.Word, 0)
		if _, err := engine.Exec("use naming;"); err != nil {
			panic(err)
		}
		if err := engine.Where("stroke_count = ?", strokeCount).Find(&words); err != nil {
			return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
		}
		unwants := make([]models.UnwantWord, 0)
		if err := engine.Where("naming_id = 1 and stroke_count = ?", strokeCount).Find(&unwants); err != nil {
			return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
		}
		unwantMap := map[int64]struct{}{}
		for _, unwant := range unwants {
			unwantMap[unwant.WordId] = struct{}{}
		}
		for i := range words {
			if _, ok := unwantMap[words[i].Id]; ok {
				words[i].Unwant = true
			}
		}
		err = c.Render(http.StatusOK, "words.html", map[string]interface{}{"Words": words})
		if err != nil {
			fmt.Println(err)
		}
		return err
	})
	e.PATCH("/unwant_names/:id", func(c echo.Context) error {
		nameID := c.Param("id")
		// try find unwant_name
		unwants := make([]models.UnwantName, 0)
		if _, err := engine.Exec("use naming;"); err != nil {
			panic(err)
		}
		if err := engine.Where("name_id = ? and naming_id = 1", nameID).Find(&unwants); err != nil {
			return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
		}
		if len(unwants) == 0 {
			unwant := models.UnwantName{
				NamingId: 1,
				NameId:   nameID,
			}
			if _, err := engine.Insert(unwant); err != nil {
				return c.String(http.StatusInternalServerError, "insert failed!\n"+err.Error())
			}
			return c.String(http.StatusOK, "created")
		}
		unwant := models.UnwantName{}
		if _, err := engine.Where("name_id = ? and naming_id = 1", nameID).Delete(&unwant); err != nil {
			return c.String(http.StatusInternalServerError, "Delete failed!\n"+err.Error())
		}
		return c.String(http.StatusOK, "removed")
	})
	e.PATCH("/unwant_words/:id", func(c echo.Context) error {
		wordID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		// try find unwant_word
		unwants := make([]models.UnwantWord, 0)
		if _, err := engine.Exec("use naming;"); err != nil {
			panic(err)
		}
		if err := engine.Where("word_id = ? and naming_id = 1", wordID).Find(&unwants); err != nil {
			return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
		}
		if len(unwants) == 0 {
			words := make([]models.Word, 0)
			if err := engine.Where("id = ?", wordID).Find(&words); err != nil {
				return c.String(http.StatusInternalServerError, "Find failed!\n"+err.Error())
			}
			unwant := models.UnwantWord{
				NamingId:    1,
				WordId:      wordID,
				StrokeCount: words[0].StrokeCount,
			}
			if _, err := engine.Insert(unwant); err != nil {
				return c.String(http.StatusInternalServerError, "insert failed!\n"+err.Error())
			}
			return c.String(http.StatusOK, "created")
		}
		unwant := models.UnwantWord{}
		if _, err := engine.Where("word_id = ? and naming_id = 1", wordID).Delete(&unwant); err != nil {
			return c.String(http.StatusInternalServerError, "Delete failed!\n"+err.Error())
		}
		return c.String(http.StatusOK, "removed")
	})
	e.Logger.Fatal(e.Start(":1323"))
}
