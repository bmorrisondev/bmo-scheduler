package scripts

import (
	"context"
	"os"
	"time"

	"github.com/bmorrisondev/bmo-scheduler/models"
	"github.com/pkg/errors"

	"github.com/bmorrisondev/go-utils"
	"github.com/dstotijn/go-notion"
	"github.com/sirupsen/logrus"
)

var SyncNotionContentTasks = models.JobWrapper{
	Name: "SyncNotionContentTasks",
	Cron: "*/2 * * * *",
	Run:  syncNotionContentTasks,
}

func syncNotionContentTasks(log *logrus.Logger) error {
	contentDbid := os.Getenv("NOTION_DB_CONTENT")
	projectsDbid := os.Getenv("NOTION_DB_PROJECTS")
	tasksDbid := os.Getenv("NOTION_DB_TASKS")
	key := os.Getenv("NOTION_API_KEY")
	// ...

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
		return errors.Wrap(err, "(SyncNotionContentTasks) query projects")
	}
	log.Infof("Found %v content projects", len(projects.Results))

	// foreach Project
	// - Get the content item
	// - Get the tasks from the project
	// - Get the tasks from the content item
	// - Compare the two and build a list of updates
	for _, proj := range projects.Results {
		props := proj.Properties.(notion.DatabasePageProperties)
		log.Infof("Checking '%v'", props["Name"].Title[0].PlainText)
		contentItems, err := client.QueryDatabase(context.Background(), contentDbid, &notion.DatabaseQuery{
			Filter: &notion.DatabaseQueryFilter{
				Property: "Project",
				Relation: &notion.RelationDatabaseQueryFilter{
					Contains: proj.ID,
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "(SyncNotionContentTasks) query content items")
		}

		// Check if the project has a content item
		if len(contentItems.Results) == 0 {
			log.Infof("No content items found")
			return nil
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
			return errors.Wrap(err, "(SyncNotionContentTasks) query tasks")
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
	log.Infof("Updating %v pages", len(pageUpdates))
	for taskId, updates := range pageUpdates {
		jstr, _ := utils.ConvertToJsonString(updates)
		log.Infof("Updating page %v: %v", taskId, jstr)
		client.UpdatePage(context.Background(), taskId, updates)
		time.Sleep(500)
	}
	return nil
}

func containsRelation(props notion.DatabasePageProperties, fieldName string, contains string) bool {
	for _, el := range props[fieldName].Relation {
		if el.ID == contains {
			return true
		}
	}
	return false
}
