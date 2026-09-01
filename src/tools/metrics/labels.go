package metrics

// CycleErrReason причина раннего выхода цикла агрегации
type CycleErrReason string

const (
	// CycleErrDirsRead не удалось прочитать список nas_ip директорий
	CycleErrDirsRead CycleErrReason = "dirs_read"
	// CycleErrNilMaps загрузочная горутина вернула nil мапку каналов или сессий
	CycleErrNilMaps CycleErrReason = "nil_maps"
)

// Phase фаза цикла агрегации
type Phase string

const (
	// PhaseReadDirs чтение списка nas_ip директорий
	PhaseReadDirs Phase = "read_dirs"
	// PhaseLoadChannels загрузка мапки каналов
	PhaseLoadChannels Phase = "load_channels"
	// PhaseLoadSessions загрузка мапки онлайн сессий
	PhaseLoadSessions Phase = "load_sessions"
)

// NASResult исход обработки одного nas_ip
type NASResult string

const (
	// NASResultOK чанки посчитаны и сохранены
	NASResultOK NASResult = "ok"
	// NASResultNoNew новых flow файлов нет, выполнена повторная очистка tmp
	NASResultNoNew NASResult = "no_new"
	// NASResultNoInternal во flow нет учитываемого internal трафика
	NASResultNoInternal NASResult = "no_internal"
	// NASResultUnrecognized flow не распознан, файлы оставлены в tmp
	NASResultUnrecognized NASResult = "unrecognized"
	// NASResultError обработка прервана ошибкой
	NASResultError NASResult = "error"
)

// NASStage этап обработки nas_ip, на котором произошла ошибка
type NASStage string

const (
	// NASStageCheckpoint загрузка чекпоинта закоммиченных flow файлов
	NASStageCheckpoint NASStage = "checkpoint"
	// NASStagePrepare подготовка flow
	NASStagePrepare NASStage = "prepare"
	// NASStageParse парсинг flow
	NASStageParse NASStage = "parse"
	// NASStageSift просеивание трафика по сессиям
	NASStageSift NASStage = "sift"
	// NASStageSave сохранение чанков и чекпоинта
	NASStageSave NASStage = "save"
)

// NASPhase фаза обработки одного nas_ip
type NASPhase string

const (
	// NASPhasePrepareFlow подготовка flow
	NASPhasePrepareFlow NASPhase = "prepare_flow"
	// NASPhaseParseFlow парсинг flow и подсчет трафика
	NASPhaseParseFlow NASPhase = "parse_flow"
	// NASPhaseSiftTraffic привязка трафика к сессиям
	NASPhaseSiftTraffic NASPhase = "sift_traffic"
	// NASPhaseSaveChunks сохранение чанков и чекпоинта в бд
	NASPhaseSaveChunks NASPhase = "save_chunks"
)

// Direction направление учтенного трафика
type Direction string

const (
	// DirectionDownload входящий трафик клиента
	DirectionDownload Direction = "download"
	// DirectionUpload исходящий трафик клиента
	DirectionUpload Direction = "upload"
)

// списки всех значений лейблов для преинициализации серий
var (
	allCycleErrReasons = []CycleErrReason{CycleErrDirsRead, CycleErrNilMaps}
	allPhases          = []Phase{PhaseReadDirs, PhaseLoadChannels, PhaseLoadSessions}
	allNASResults      = []NASResult{
		NASResultOK, NASResultNoNew, NASResultNoInternal, NASResultUnrecognized, NASResultError,
	}
	allNASStages = []NASStage{
		NASStageCheckpoint, NASStagePrepare, NASStageParse, NASStageSift, NASStageSave,
	}
	allNASPhases = []NASPhase{
		NASPhasePrepareFlow, NASPhaseParseFlow, NASPhaseSiftTraffic, NASPhaseSaveChunks,
	}
	allDirections = []Direction{DirectionDownload, DirectionUpload}
)
