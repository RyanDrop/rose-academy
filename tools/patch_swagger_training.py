"""One-off merge of training API paths into docs/swagger.json. Run: py -3 tools/patch_swagger_training.py"""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
p = ROOT / "docs" / "swagger.json"
d = json.loads(p.read_text(encoding="utf-8"))

obj = {"type": "object", "additionalProperties": True}


def bearer():
    return [{"BearerAuth": []}]


new_paths = {
    "/exercises": {
        "get": {
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Listar exercicios",
            "parameters": [
                {
                    "type": "string",
                    "description": "filtro ILIKE em nome, musculo alvo, equipamento",
                    "name": "q",
                    "in": "query",
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {
                        "type": "array",
                        "items": {"$ref": "#/definitions/models.ExercisePublic"},
                    },
                },
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Meses do plano ativo",
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {
                        "type": "array",
                        "items": {"$ref": "#/definitions/models.UserTrainingMonthItem"},
                    },
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months/{month_id}/progress/reset": {
        "post": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Resetar progresso do mes",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "month_id",
                    "in": "path",
                    "required": True,
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.ResetMonthProgressResponse"},
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months/{month_id}/progression-summary": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Resumo comparativo mensal por exercicio",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "month_id",
                    "in": "path",
                    "required": True,
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {
                        "type": "array",
                        "items": {"$ref": "#/definitions/models.MonthlyProgressionSummaryRow"},
                    },
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months/{month_id}/statistics": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Estatisticas agregadas do mes",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "month_id",
                    "in": "path",
                    "required": True,
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.MonthStatisticsRow"},
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months/{month_id}/weight-progression": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Progressao de cargas no mes",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "month_id",
                    "in": "path",
                    "required": True,
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {
                        "type": "array",
                        "items": {"$ref": "#/definitions/models.WeightProgressionRow"},
                    },
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months/{month_id}/weekly-performance": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Melhor desempenho semanal por exercicio",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "month_id",
                    "in": "path",
                    "required": True,
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {
                        "type": "array",
                        "items": {"$ref": "#/definitions/models.WeeklyBestPerformanceRow"},
                    },
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months/{month_id}/workouts": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Treinos do mes (get_month_workouts)",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "month_id",
                    "in": "path",
                    "required": True,
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {
                        "type": "array",
                        "items": {"$ref": "#/definitions/models.MonthWorkoutSummary"},
                    },
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/months/{month_number}/start": {
        "post": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Iniciar mes do plano (start_month_progress)",
            "parameters": [
                {
                    "type": "integer",
                    "name": "month_number",
                    "in": "path",
                    "required": True,
                    "description": "1-12",
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.StartMonthProgressResponse"},
                },
                "400": {"description": "Bad Request", "schema": obj},
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden (mes nao liberado pelo cronograma do plano atual)", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/plan": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Plano de treino ativo",
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.UserActivePlanResponse"},
                },
                "401": {"description": "Unauthorized", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        },
        "post": {
            "security": bearer(),
            "consumes": ["application/json"],
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Escolher plano de treino (select_training_plan)",
            "parameters": [
                {
                    "name": "body",
                    "in": "body",
                    "required": True,
                    "schema": {"$ref": "#/definitions/controllers.SelectTrainingPlanRequest"},
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.SelectTrainingPlanResponse"},
                },
                "400": {"description": "Bad Request", "schema": obj},
                "401": {"description": "Unauthorized", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "422": {"description": "Validation Failed", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        },
    },
    "/me/training/workout-completions/{completion_id}/sets": {
        "post": {
            "security": bearer(),
            "consumes": ["application/json"],
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Registrar serie (log_exercise_set)",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "completion_id",
                    "in": "path",
                    "required": True,
                },
                {
                    "name": "body",
                    "in": "body",
                    "required": True,
                    "schema": {"$ref": "#/definitions/models.LogExerciseSetRequest"},
                },
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.LogExerciseSetResponse"},
                },
                "400": {"description": "Bad Request", "schema": obj},
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "422": {"description": "Validation Failed", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/workouts/day/{day_number}": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Treino do dia (1-7); query opcional month_id",
            "parameters": [
                {"type": "integer", "name": "day_number", "in": "path", "required": True},
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "month_id",
                    "in": "query",
                    "required": False,
                },
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.WorkoutDetailResponse"},
                },
                "400": {"description": "Bad Request", "schema": obj},
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/workouts/{workout_id}": {
        "get": {
            "security": bearer(),
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Detalhe do treino",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "workout_id",
                    "in": "path",
                    "required": True,
                }
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.WorkoutDetailResponse"},
                },
                "400": {"description": "Bad Request", "schema": obj},
                "401": {"description": "Unauthorized", "schema": obj},
                "404": {"description": "Not Found", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
    "/me/training/workouts/{workout_id}/complete": {
        "post": {
            "security": bearer(),
            "consumes": ["application/json"],
            "produces": ["application/json"],
            "tags": ["training"],
            "summary": "Concluir treino (complete_workout)",
            "parameters": [
                {
                    "type": "string",
                    "format": "uuid",
                    "name": "workout_id",
                    "in": "path",
                    "required": True,
                },
                {
                    "name": "body",
                    "in": "body",
                    "required": True,
                    "schema": {"$ref": "#/definitions/models.CompleteWorkoutRequest"},
                },
            ],
            "responses": {
                "200": {
                    "description": "OK",
                    "schema": {"$ref": "#/definitions/models.CompleteWorkoutResponse"},
                },
                "400": {"description": "Bad Request", "schema": obj},
                "401": {"description": "Unauthorized", "schema": obj},
                "403": {"description": "Forbidden", "schema": obj},
                "409": {"description": "Conflict", "schema": obj},
                "422": {"description": "Validation Failed", "schema": obj},
                "500": {"description": "Internal Server Error", "schema": obj},
            },
        }
    },
}

new_defs = {
    "controllers.SelectTrainingPlanRequest": {
        "type": "object",
        "properties": {
            "level": {
                "type": "string",
                "description": "iniciante, intermediario ou avancado (aceita variantes)",
            }
        },
    },
    "models.CompleteWorkoutRequest": {
        "type": "object",
        "properties": {
            "week_number": {"type": "integer", "description": "semana do mes 1-4"},
            "notes": {"type": "string"},
        },
    },
    "models.CompleteWorkoutResponse": {
        "type": "object",
        "properties": {"completion_id": {"type": "string", "format": "uuid"}},
    },
    "models.ExercisePublic": {
        "type": "object",
        "properties": {
            "id": {"type": "string", "format": "uuid"},
            "name": {"type": "string"},
            "equipment": {"type": "string"},
            "target_muscle": {"type": "string"},
            "video_url": {"type": "string"},
            "video_embed_url": {"type": "string"},
            "instructions": {"type": "string"},
            "thumbnail_url": {"type": "string"},
        },
    },
    "models.LogExerciseSetRequest": {
        "type": "object",
        "properties": {
            "workout_exercise_id": {"type": "string", "format": "uuid"},
            "set_number": {"type": "integer"},
            "weight_used": {"type": "number"},
            "reps_completed": {"type": "integer"},
            "notes": {"type": "string"},
        },
    },
    "models.LogExerciseSetResponse": {
        "type": "object",
        "properties": {"set_id": {"type": "string", "format": "uuid"}},
    },
    "models.MonthStatisticsRow": {
        "type": "object",
        "properties": {
            "total_workouts_completed": {"type": "integer"},
            "total_exercises_completed": {"type": "integer"},
            "total_sets_completed": {"type": "integer"},
            "average_weight": {"type": "number"},
            "max_weight": {"type": "number"},
            "total_volume": {"type": "number"},
            "weeks_completed": {"type": "integer"},
            "completion_percentage": {"type": "number"},
        },
    },
    "models.MonthWorkoutSummary": {
        "type": "object",
        "properties": {
            "workout_id": {"type": "string", "format": "uuid"},
            "day_number": {"type": "integer"},
            "day_name": {"type": "string"},
            "title": {"type": "string"},
            "total_exercises": {"type": "integer", "format": "int64"},
        },
    },
    "models.MonthlyProgressionSummaryRow": {
        "type": "object",
        "properties": {
            "exercise_name": {"type": "string"},
            "equipment": {"type": "string"},
            "target_muscle": {"type": "string"},
            "week1_weight": {"type": "number"},
            "week2_weight": {"type": "number"},
            "week3_weight": {"type": "number"},
            "week4_weight": {"type": "number"},
            "initial_weight": {"type": "number"},
            "final_weight": {"type": "number"},
            "weight_gain": {"type": "number"},
            "percentage_gain": {"type": "number"},
        },
    },
    "models.ResetMonthProgressResponse": {
        "type": "object",
        "properties": {"reset": {"type": "boolean"}},
    },
    "models.SelectTrainingPlanResponse": {
        "type": "object",
        "properties": {"selection_id": {"type": "string", "format": "uuid"}},
    },
    "models.StartMonthProgressResponse": {
        "type": "object",
        "properties": {"progress_id": {"type": "string", "format": "uuid"}},
    },
    "models.UserActivePlanResponse": {
        "type": "object",
        "properties": {
            "plan_id": {"type": "string", "format": "uuid"},
            "level": {"type": "string"},
            "plan_name": {"type": "string"},
            "selected_at": {"type": "string", "format": "date-time"},
        },
    },
    "models.UserTrainingMonthItem": {
        "type": "object",
        "properties": {
            "month_id": {"type": "string", "format": "uuid"},
            "month_number": {"type": "integer"},
            "title": {"type": "string"},
            "description": {"type": "string"},
            "is_started": {"type": "boolean"},
            "is_completed": {"type": "boolean"},
            "started_at": {"type": "string", "format": "date-time"},
            "completed_at": {"type": "string", "format": "date-time"},
            "active": {
                "type": "boolean",
                "description": "Sem mês iniciado: só mês 1; depois selected_at + calendário (1..12); calculado na API",
            },
        },
    },
    "models.WeightProgressionRow": {
        "type": "object",
        "properties": {
            "exercise_name": {"type": "string"},
            "equipment": {"type": "string"},
            "target_muscle": {"type": "string"},
            "week_number": {"type": "integer"},
            "set_number": {"type": "integer"},
            "weight_used": {"type": "number"},
            "reps_completed": {"type": "integer"},
            "completed_at": {"type": "string"},
        },
    },
    "models.WeeklyBestPerformanceRow": {
        "type": "object",
        "properties": {
            "exercise_id": {"type": "string", "format": "uuid"},
            "exercise_name": {"type": "string"},
            "equipment": {"type": "string"},
            "target_muscle": {"type": "string"},
            "week_number": {"type": "integer"},
            "max_weight": {"type": "number"},
            "avg_weight": {"type": "number"},
            "total_sets": {"type": "integer"},
            "avg_reps": {"type": "number"},
            "last_completed": {"type": "string"},
        },
    },
    "models.WorkoutDetailResponse": {
        "type": "object",
        "properties": {
            "workout": {"$ref": "#/definitions/models.WorkoutPublic"},
            "exercises": {
                "type": "array",
                "items": {"$ref": "#/definitions/models.WorkoutExercisePublic"},
            },
        },
    },
    "models.WorkoutExercisePublic": {
        "type": "object",
        "properties": {
            "workout_exercise_id": {"type": "string", "format": "uuid"},
            "order_position": {"type": "integer"},
            "sets": {"type": "integer"},
            "reps": {"type": "string"},
            "rest_time_seconds": {"type": "integer"},
            "instructions": {"type": "string"},
            "notes": {"type": "string"},
            "exercise": {"$ref": "#/definitions/models.ExercisePublic"},
        },
    },
    "models.WorkoutPublic": {
        "type": "object",
        "properties": {
            "id": {"type": "string", "format": "uuid"},
            "title": {"type": "string"},
            "description": {"type": "string"},
            "day_number": {"type": "integer"},
            "day_name": {"type": "string"},
        },
    },
}


def main():
    d["paths"].update(new_paths)
    d["definitions"].update(new_defs)
    if "securityDefinitions" not in d:
        d["securityDefinitions"] = {
            "BearerAuth": {"type": "apiKey", "name": "Authorization", "in": "header"}
        }
    d["paths"] = {k: d["paths"][k] for k in sorted(d["paths"].keys())}
    d["definitions"] = {k: d["definitions"][k] for k in sorted(d["definitions"].keys())}
    p.write_text(json.dumps(d, indent=4, ensure_ascii=True) + "\n", encoding="utf-8")
    print("OK paths=", len(d["paths"]), "definitions=", len(d["definitions"]))
    patch_docs_go(d)


def patch_docs_go(sw: dict) -> None:
    """Atualiza docTemplate em docs/docs.go (Swagger UI usa o embed, nao swagger.json)."""
    dg_path = ROOT / "docs" / "docs.go"
    dg = dg_path.read_text(encoding="utf-8")
    start = dg.index("const docTemplate = `") + len("const docTemplate = `")
    end = dg.index("`\n\n// SwaggerInfo")
    old_inner = dg[start:end]

    i = old_inner.index('"paths": {\n') + len('"paths": {\n')
    pre_paths = old_inner[:i]

    def inner_block(obj: dict) -> str:
        raw = json.dumps(obj, indent=4, ensure_ascii=True)
        lines = raw.split("\n")[1:-1]
        return "\n".join("    " + line for line in lines)

    paths_body = inner_block(sw["paths"])
    defs_body = inner_block(sw["definitions"])

    tail = "\n    }\n}"
    if sw.get("securityDefinitions"):
        tail = (
            "\n    },\n    \"securityDefinitions\": "
            + json.dumps(sw["securityDefinitions"], ensure_ascii=True)
            + "\n}"
        )

    new_inner = (
        pre_paths
        + paths_body
        + "\n    },\n    \"definitions\": {\n"
        + defs_body
        + tail
    )

    new_dg = dg[:start] + new_inner + dg[end:]
    dg_path.write_text(new_dg, encoding="utf-8")
    print("OK docs/docs.go docTemplate atualizado")


if __name__ == "__main__":
    main()
