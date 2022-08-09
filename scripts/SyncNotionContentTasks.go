package scripts

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/bmorrisondev/go-utils"
	"github.com/dstotijn/go-notion"
	gocron "github.com/go-co-op/gocron"
)

func SyncNotionContentTasks(s *gocron.Scheduler) {
	contentDbid := os.Getenv("NOTION_DB_CONTENT")
	projectsDbid := os.Getenv("NOTION_DB_PROJECTS")
	tasksDbid := os.Getenv("NOTION_DB_TASKS")
	key := os.Getenv("NOTION_API_KEY")

	s.Every(2).Minute().Do(func() {
		log.Println("Running \"SyncNotionContentTasks\"")
		client := notion.NewClient(key)

		pageUpdates := map[string]notion.UpdatePageParams{}

		// Get a list of active projects, where project type is "Content"
		projects, err := client.QueryDatabase(context.Background(), projectsDbid, &notion.DatabaseQuery{
			Filter: &notion.DatabaseQueryFilter{
				Property: "Type",
				Select: &notion.SelectDatabaseQueryFilter{
					Equals: "Content",
				},
			},
		})
		if err != nil {
			log.Println("(SyncNotionContentTasks) query database failed:", err)
			return
		}

		// foreach Project
		// - Get the content item
		// - Get the tasks from the project
		// - Get the tasks from the content item
		// - Compare the two and build a list of updates
		for _, proj := range projects.Results {
			contentItems, err := client.QueryDatabase(context.Background(), contentDbid, &notion.DatabaseQuery{
				Filter: &notion.DatabaseQueryFilter{
					Property: "Project",
					Relation: &notion.RelationDatabaseQueryFilter{
						Contains: proj.ID,
					},
				},
			})
			if err != nil {
				log.Println(err)
				return
			}

			// Check if the project has a content item
			if len(contentItems.Results) == 0 {
				return
			}

			ci := contentItems.Results[0]

			tasks, err := client.QueryDatabase(context.Background(), tasksDbid, &notion.DatabaseQuery{
				Filter: &notion.DatabaseQueryFilter{
					Or: []notion.DatabaseQueryFilter{
						{
							Property: "Content Item",
							Relation: &notion.RelationDatabaseQueryFilter{
								Contains: ci.ID,
							},
						},
						{
							Property: "Project",
							Relation: &notion.RelationDatabaseQueryFilter{
								Contains: proj.ID,
							},
						},
					},
				},
			})
			if err != nil {
				log.Println(err)
				return
			}

			for _, task := range tasks.Results {
				requiresUpdate := false
				updates := notion.UpdatePageParams{
					DatabasePageProperties: notion.DatabasePageProperties{},
				}

				taskProps := task.Properties.(notion.DatabasePageProperties)
				if !containsRelation(taskProps, "Content Item", ci.ID) {
					updates.DatabasePageProperties["Content Item"] = notion.DatabasePageProperty{
						Relation: []notion.Relation{
							{
								ID: ci.ID,
							},
						},
					}
					requiresUpdate = true
				}

				if !containsRelation(taskProps, "Project", proj.ID) {
					updates.DatabasePageProperties["Project"] = notion.DatabasePageProperty{
						Relation: []notion.Relation{
							{
								ID: proj.ID,
							},
						},
					}
					requiresUpdate = true
				}

				if requiresUpdate {
					pageUpdates[task.ID] = updates
				}
			}
		}

		// foreach Update
		// - Execute a page update op, making sure the
		for taskId, updates := range pageUpdates {
			jstr, _ := utils.ConvertToJsonString(updates)
			log.Printf("Updating page %v: %v", taskId, jstr)
			client.UpdatePage(context.Background(), taskId, updates)
			time.Sleep(500)
		}
	})
}

func containsRelation(props notion.DatabasePageProperties, fieldName string, contains string) bool {
	for _, el := range props[fieldName].Relation {
		if el.ID == contains {
			return true
		}
	}
	return false
}
