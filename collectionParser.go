package modelbind

import (
	"reflect"
	"strings"

	"github.com/dronm/modelbind/metadata"
	"github.com/dronm/modelbind/pg"
	"github.com/dronm/modelbind/types"
)

// ParseLimitParams transfers offset/count values from CollectionParams
// into the provided DBLimit builder.
func ParseLimitParams(dbLimit types.DBLimit, params CollectionParams) error {
	const funcName = "ParseLimitParams"

	if dbLimit == nil {
		return NewMessageError(
			MsgLimitNotInitialized,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	dbLimit.SetFrom(int(params.From))
	dbLimit.SetCount(int(params.Count))

	return nil
}

// ParseSorterParams validates sorter fields and directions from CollectionParams,
// checks that referenced fields exist in the model metadata, sanitizes SQL field
// expressions, and adds the resulting sort rules to the provided DBSorters builder.
func ParseSorterParams(model types.DBModel, dbSorter types.DBSorters, params CollectionParams) error {
	const funcName = "ParseSorterParams"

	modelMd, err := metadata.NewModelMetadata(model)
	if err != nil {
		return err
	}

	for _, sorter := range params.Sorter {
		fieldID := sorter.Field
		structFieldInd := strings.Index(fieldID, "->")
		if structFieldInd >= 0 {
			fieldID = fieldID[:structFieldInd]
		}

		if _, ok := modelMd.Fields[fieldID]; !ok {
			return NewMessageError(
				MsgMetadataFieldNotFound,
				map[string]any{
					"Func":  funcName,
					"Field": fieldID,
				},
			)
		}

		if _, err := pg.SanitizeSQLFieldRef(sorter.Field); err != nil {
			return NewMessageError(
				MsgInvalidFieldExpression,
				map[string]any{
					"Func":       funcName,
					"Expression": sorter.Field,
				},
			)
		}

		var sortDirect types.SQLSortDirect
		switch sorter.Direct {
		case SortParDesc:
			sortDirect = types.SQLSortDesc
		default:
			sortDirect = types.SQLSortAsc
		}

		if dbSorter == nil {
			return NewMessageError(
				MsgSorterNotInitialized,
				map[string]any{
					"Func": funcName,
				},
			)
		}

		dbSorter.Add(sorter.Field, sortDirect)
	}

	return nil
}

// ParseFilterParams validates filter fields, values, operators, and joins from
// CollectionParams, checks that referenced fields exist in the model metadata,
// sanitizes SQL field expressions, and adds the resulting filters to the provided
// DBFilters builder.
func ParseFilterParams(model types.DBModel, dbFilter types.DBFilters, params CollectionParams, table string) error {
	const funcName = "ParseFilterParams"

	modelMd, err := metadata.NewModelMetadata(model)
	if err != nil {
		return err
	}

	validationErr := &ValidationError{}

	for _, filter := range params.Filter {
		var join types.FilterJoin
		switch filter.Join {
		case FilterParJoinOr:
			join = types.SQLFilterJoinOr
		default:
			join = types.SQLFilterJoinAnd
		}

		for filterFieldID, filterField := range filter.Fields {
			filterModelFieldID := filterFieldID

			structFieldInd := strings.Index(filterModelFieldID, "->")
			if structFieldInd >= 0 {
				filterModelFieldID = filterModelFieldID[:structFieldInd]
			}

			if _, err := pg.SanitizeSQLFieldRef(filterFieldID); err != nil {
				return NewMessageError(
					MsgInvalidFieldExpression,
					map[string]any{
						"Func":       funcName,
						"Expression": filterFieldID,
					},
				)
			}

			fieldMd, ok := modelMd.Fields[filterModelFieldID]
			if !ok {
				return NewMessageError(
					MsgMetadataFieldNotFound,
					map[string]any{
						"Func":  funcName,
						"Field": filterModelFieldID,
					},
				)
			}

			if filterField.Value != nil {
				if _, err := fieldMd.Validate(reflect.ValueOf(filterField.Value)); err != nil {
					validationErr.Add(err)
					continue
				}
			}

			var operator types.SQLFilterOperator
			switch filterField.Operator {
			case FilterOperParEq:
				operator = types.SQLFilterOperatorEq
			case FilterOperParLess:
				operator = types.SQLFilterOperatorLess
			case FilterOperParGr:
				operator = types.SQLFilterOperatorGr
			case FilterOperParLessEq:
				operator = types.SQLFilterOperatorLessEq
			case FilterOperParGrEq:
				operator = types.SQLFilterOperatorGrEq
			case FilterOperParLk:
				operator = types.SQLFilterOperatorLk
			case FilterOperParILk:
				operator = types.SQLFilterOperatorILk
			case FilterOperParNotEq:
				operator = types.SQLFilterOperatorNotEq
			case FilterOperParIs:
				operator = types.SQLFilterOperatorIs
			case FilterOperParIn:
				operator = types.SQLFilterOperatorIn
			case FilterOperParIncl:
				operator = types.SQLFilterOperatorIncl
			case FilterOperParAny:
				operator = types.SQLFilterOperatorAny
			case FilterOperParHas:
				operator = types.SQLFilterOperatorHas
			case FilterOperParOverlap:
				operator = types.SQLFilterOperatorOverlap
			case FilterOperParContains:
				operator = types.SQLFilterOperatorContains
			case FilterOperParTS:
				operator = types.SQLFilterOperatorTS
			}

			if dbFilter == nil {
				return NewMessageError(
					MsgFilterNotInitialized,
					map[string]any{
						"Func": funcName,
					},
				)
			}

			switch operator {
			case types.SQLFilterOperatorTS:
				dbFilter.AddFullTextSearch(table, filterFieldID, filterField.Value, join)
			case types.SQLFilterOperatorIncl:
				dbFilter.AddArrayInclude(table, filterFieldID, filterField.Value, join)
			case types.SQLFilterOperatorHas:
				dbFilter.AddColumnArrayInclude(table, filterFieldID, filterField.Value, join)
			default:
				dbFilter.Add(table, filterFieldID, filterField.Value, operator, join)
			}
		}
	}

	if validationErr.HasErrors() {
		return validationErr
	}

	return nil
}
