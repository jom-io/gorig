package test

import (
	"context"
	"github.com/jom-io/gorig/domainx"
	"github.com/jom-io/gorig/domainx/dx"
	"github.com/jom-io/gorig/serv"
	"gorm.io/datatypes"
	"testing"
	"time"
)

type TestModel struct {
	TestField1 string         `gorm:"column:test_field1;type:varchar(255);comment:test_field1" bson:"test_field1" json:"testField1" form:"testField1"`
	TestField2 int            `gorm:"column:test_field2;type:int;comment:test_field2" bson:"test_field2" json:"testField2" form:"testField2"`
	TestField3 bool           `gorm:"column:test_field3;type:tinyint(1);comment:test_field3" bson:"test_field3" json:"testField3" form:"testField3"`
	TestField4 float64        `gorm:"column:test_field4;type:decimal(10,2);comment:test_field4" bson:"test_field4" json:"testField4" form:"testField4"`
	TestField5 datatypes.JSON `gorm:"column:test_field5;type:json;comment:test_field5" bson:"test_field5" json:"testField5" form:"testField5"`
}

// Mysql configuration for the TestModel
func (t *TestModel) DConfig() (domainx.ConType, string, string) {
	return domainx.Mysql, "main", "test_model"
}

// Mongo configuration for the TestModel
//func (t *TestModel) DConfig() (domainx.ConType, string, string) {
//	return domainx.Mongo, "main", "test_model"
//}

func setupTestModel() *TestModel {
	return &TestModel{
		TestField1: "example",
		TestField2: 42,
		TestField3: true,
		TestField4: 3.14,
		TestField5: []byte(`["A", "B", "C"]`),
	}
}

func TestTestModel_CRUD(t *testing.T) {
	ctx := context.Background()
	domainx.AutoMigrate(func() (value domainx.ConTable) {
		return dx.On[TestModel](ctx).Complex()
	})

	if codeErr := serv.StartCode(domainx.ServiceCode); codeErr != nil {
		panic(codeErr)
	}
	time.Sleep(3 * time.Second)

	var id int64

	t.Run("Init", func(t *testing.T) {
		data := dx.On(ctx, setupTestModel()).GetData()
		if data == nil {
			t.Fatal("Failed to initialize TestModel data")
		}
		t.Logf("Initialized TestModel: %+v", data)
	})

	t.Run("Save", func(t *testing.T) {
		model := setupTestModel()
		nID, err := dx.On(ctx, model).Save()
		id = nID
		if err != nil {
			t.Fatalf("Failed to save TestModel: %v", err)
		}
		if id <= 0 {
			t.Fatalf("Expected valid ID after save, got: %d", id)
		}
		t.Logf("Saved TestModel ID: %d", id)
	})

	t.Run("UpdateWithID", func(t *testing.T) {
		err := dx.On[TestModel](ctx).WithID(id).Update("test_field2", 100)
		if err != nil {
			t.Fatalf("Failed to update field2: %v", err)
		}
	})

	t.Run("UpdateWithMap", func(t *testing.T) {
		err := dx.On[TestModel](ctx).Eq("id", id).Updates(map[string]interface{}{
			"test_field2": 200,
		})
		if err != nil {
			t.Fatalf("Failed to update with map: %v", err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		result, err := dx.On[TestModel](ctx).WithID(id).Get()
		if err != nil {
			t.Fatalf("Failed to get model: %v", err)
		}
		if result.IsNil() {
			t.Fatal("Get returned nil result")
		}
		t.Logf("Retrieved model: %+v", result.Data)
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := dx.On[TestModel](ctx).Eq("test_field1", "example").Exists()
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if !exists {
			t.Fatal("Expected model to exist")
		}
		t.Logf("Model exists: %v", exists)

		existsFalse, err := dx.On[TestModel](ctx).Eq("test_field1", "nonexistent").Exists()
		if err != nil {
			t.Fatalf("Failed to check non-existence: %v", err)
		}
		if existsFalse {
			t.Fatal("Expected model to not exist")
		}

		existsByID, err := dx.On[TestModel](ctx).WithID(id).Exists()
		if err != nil {
			t.Fatalf("Failed to check existence by ID: %v", err)
		}
		if !existsByID {
			t.Fatal("Expected model to exist by ID")
		}
		t.Logf("Model exists by ID: %v", existsByID)
	})

	t.Run("Find", func(t *testing.T) {
		results, err := dx.On[TestModel](ctx).Eq("test_field1", "example").Sort("id").Find()
		if err != nil {
			t.Fatalf("Failed to find models: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Expected non-empty results from Find")
		}
		for _, m := range results.List() {
			t.Logf("Found model: %+v", m)
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := dx.On[TestModel](ctx).Count()
		if err != nil {
			t.Fatalf("Failed to count models: %v", err)
		}
		t.Logf("Model count: %d", count)
	})

	t.Run("Sum", func(t *testing.T) {
		sum, err := dx.On[TestModel](ctx).Sum("test_field4")
		if err != nil {
			t.Fatalf("Failed to sum field: %v", err)
		}
		t.Logf("Sum of field4: %f", sum)
	})

	t.Run("Page", func(t *testing.T) {
		pageResp, err := dx.On[TestModel](ctx).Page(1, 2, 0)
		if err != nil {
			t.Fatalf("Failed to page models: %v", err)
		}
		if pageResp == nil || pageResp.Result == nil {
			t.Fatal("Expected non-nil PageResp and Result")
		}
		t.Logf("Page info: Page %d, Size %d, Total %d, LastID %d", pageResp.Page, pageResp.Size, pageResp.Total.Get(), pageResp.LastID)
		for _, m := range *pageResp.Result {
			t.Logf("Paged model: %+v", m.Data)
		}
	})

	t.Run("Has", func(t *testing.T) {
		results, err := dx.On[TestModel](ctx).Has("test_field5", "A").Find()
		if err != nil {
			t.Fatalf("Failed to find models with Has: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Expected non-empty results from Has")
		}
		for _, m := range results.List() {
			t.Logf("Has model: %+v", m)
		}
		resultsB, err := dx.On[TestModel](ctx).Has("test_field5", "D").Find()
		if err != nil {
			t.Fatalf("Failed to find models with Has: %v", err)
		}
		if len(resultsB) != 0 {
			t.Fatal("Expected empty results from Has for non-existing value")
		}
	})

	t.Run("HasAny", func(t *testing.T) {
		results, err := dx.On[TestModel](ctx).HasAny("test_field5", []string{"B", "C"}).Find()
		if err != nil {
			t.Fatalf("Failed to find models with HasAny: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Expected non-empty results from HasAny")
		}
		for _, m := range results.List() {
			t.Logf("HasAny model: %+v", m)
		}
		resultsB, err := dx.On[TestModel](ctx).HasAny("test_field5", []string{"D", "E"}).Find()
		if err != nil {
			t.Fatalf("Failed to find models with HasAny: %v", err)
		}
		if len(resultsB) != 0 {
			t.Fatal("Expected empty results from HasAny for non-existing values")
		}
	})

	t.Run("HasAll", func(t *testing.T) {
		results, err := dx.On[TestModel](ctx).HasAll("test_field5", []string{"A", "C"}).Find()
		if err != nil {
			t.Fatalf("Failed to find models with HasAll: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Expected non-empty results from HasAll")
		}
		for _, m := range results.List() {
			t.Logf("HasAll model: %+v", m)
		}

		resultsB, err := dx.On[TestModel](ctx).HasAll("test_field5", []string{"A", "D"}).Find()
		if err != nil {
			t.Fatalf("Failed to find models with HasAll: %v", err)
		}
		if len(resultsB) != 0 {
			t.Fatal("Expected empty results from HasAll for non-existing combination")
		}
	})

	t.Run("DeleteByID", func(t *testing.T) {
		err := dx.On[TestModel](ctx).WithID(id).Delete()
		if err != nil {
			t.Fatalf("Failed to delete model by ID: %v", err)
		}
		t.Logf("Deleted model ID: %d", id)
	})

	t.Run("DeleteByMatch", func(t *testing.T) {
		err := dx.On[TestModel](ctx).Eq("test_field1", "example").Delete()
		if err != nil {
			t.Fatalf("Failed to delete model by match: %v", err)
		}
	})
}
