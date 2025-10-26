package entity

import (
	"mkanban/internal/domain/valueobject"
	"strings"
)

// ConditionOperator represents comparison operators
type ConditionOperator string

const (
	OperatorEquals       ConditionOperator = "eq"
	OperatorNotEquals    ConditionOperator = "ne"
	OperatorContains     ConditionOperator = "contains"
	OperatorNotContains  ConditionOperator = "not_contains"
	OperatorGreaterThan  ConditionOperator = "gt"
	OperatorLessThan     ConditionOperator = "lt"
	OperatorIn           ConditionOperator = "in"
	OperatorNotIn        ConditionOperator = "not_in"
)

// Condition represents a filtering condition for actions
type Condition struct {
	Field    string            // e.g., "priority", "status", "tags", "column"
	Operator ConditionOperator
	Value    interface{} // The value to compare against
}

// Evaluate evaluates the condition against a task
func (c *Condition) Evaluate(task *Task, column *Column) bool {
	var actualValue interface{}

	// Extract the field value from the task or column
	switch c.Field {
	case "priority":
		actualValue = task.Priority().String()
	case "status":
		actualValue = task.Status().String()
	case "column":
		if column != nil {
			actualValue = column.Name()
		}
	case "tags":
		actualValue = task.Tags()
	case "has_due_date":
		actualValue = task.DueDate() != nil
	case "is_overdue":
		actualValue = task.IsOverdue()
	default:
		// Check metadata
		if val, exists := task.GetMetadata(c.Field); exists {
			actualValue = val
		} else {
			return false
		}
	}

	return c.compareValues(actualValue)
}

// compareValues compares the actual value with the condition value using the operator
func (c *Condition) compareValues(actualValue interface{}) bool {
	switch c.Operator {
	case OperatorEquals:
		return actualValue == c.Value
	case OperatorNotEquals:
		return actualValue != c.Value
	case OperatorContains:
		if tags, ok := actualValue.([]string); ok {
			if val, ok := c.Value.(string); ok {
				return containsString(tags, val)
			}
		}
		if str, ok := actualValue.(string); ok {
			if val, ok := c.Value.(string); ok {
				return strings.Contains(str, val)
			}
		}
		return false
	case OperatorNotContains:
		return !c.compareValues(actualValue)
	case OperatorIn:
		if values, ok := c.Value.([]string); ok {
			if str, ok := actualValue.(string); ok {
				return containsString(values, str)
			}
		}
		return false
	case OperatorNotIn:
		if values, ok := c.Value.([]string); ok {
			if str, ok := actualValue.(string); ok {
				return !containsString(values, str)
			}
		}
		return false
	default:
		return false
	}
}

// containsString checks if a slice contains a string
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// ConditionGroup represents a group of conditions with logical operators
type ConditionGroup struct {
	Conditions []*Condition
	Operator   LogicalOperator // AND or OR
}

// LogicalOperator represents logical operators for combining conditions
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "AND"
	LogicalOr  LogicalOperator = "OR"
)

// Evaluate evaluates all conditions in the group
func (cg *ConditionGroup) Evaluate(task *Task, column *Column) bool {
	if len(cg.Conditions) == 0 {
		return true // No conditions means always true
	}

	switch cg.Operator {
	case LogicalAnd:
		for _, condition := range cg.Conditions {
			if !condition.Evaluate(task, column) {
				return false
			}
		}
		return true
	case LogicalOr:
		for _, condition := range cg.Conditions {
			if condition.Evaluate(task, column) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// NewCondition creates a new condition
func NewCondition(field string, operator ConditionOperator, value interface{}) *Condition {
	return &Condition{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
}

// NewConditionGroup creates a new condition group
func NewConditionGroup(operator LogicalOperator, conditions ...*Condition) *ConditionGroup {
	return &ConditionGroup{
		Conditions: conditions,
		Operator:   operator,
	}
}
