# FRD-20251030-007: Offline Context Adaptation

**Feature:** ACE Feature 7 - Offline Context Adaptation  
**Status:** Draft  
**Created:** 2025-10-30  
**Author:** Spin Development Team  
**Depends On:** Feature 1 (Core), Feature 2 (Generator), Feature 3 (Reflector), Feature 4 (Curator)

---

## Executive Summary

Implement **offline context adaptation** where ACE playbooks are refined using training data before deployment. Unlike online adaptation (Feature 8) which learns during task execution, offline adaptation optimizes playbooks in a separate training phase using historical task trajectories and outcomes.

**Key Innovation:** Enable pre-deployment playbook optimization through iterative refinement over training datasets, similar to supervised learning but for context engineering.

**Primary Goal:** Train high-quality playbooks that improve agent performance on downstream tasks without requiring online learning.

---

## Background

### Problem Statement

ACE playbooks can be manually seeded with domain knowledge or grown organically through online adaptation, but both approaches have limitations:

1. **Manual Seeding**: Requires expert knowledge, time-consuming, may miss important patterns
2. **Online-Only Learning**: Requires many task executions to converge, risks low-quality bullets during early phases
3. **Cold Start**: New agents start with empty or minimal playbooks, leading to poor initial performance

**Offline adaptation solves this** by training playbooks on existing task data before deployment, enabling agents to start with high-quality contexts.

### Use Cases

1. **Pre-Training Go Agent**: Train playbook on historical Go coding tasks and best practices
2. **Domain Specialization**: Create specialized playbooks for domains (security, performance, testing)
3. **Playbook Transfer**: Train on one codebase, deploy to similar codebases
4. **Continuous Improvement**: Periodically retrain playbooks on accumulated production data
5. **A/B Testing**: Train multiple playbook variants and compare performance

---

## Goals

### Functional Goals

1. **Training Loop**: Iterative refinement over training dataset (multi-epoch support)
2. **Convergence Detection**: Automatic stopping when playbook quality plateaus
3. **Validation Pipeline**: Measure playbook performance on held-out validation set
4. **Checkpointing**: Save/resume training state for long training runs
5. **Baseline Comparison**: Compare trained playbook against baseline (empty, manual-seed)

### Non-Functional Goals

1. **Efficiency**: Train on 1000+ samples in reasonable time (<1 hour)
2. **Reproducibility**: Deterministic training given same data and seed
3. **Observability**: Track metrics per epoch (playbook size, validation accuracy)
4. **Flexibility**: Support different training configurations (epochs, batch size, learning rate analog)

---

## Technical Design

### Architecture

```
internal/ace/offline/
├── trainer.go           # OfflineTrainer main component
├── training_data.go     # TrainingDataset, TrainingSample types
├── epoch.go             # Epoch execution logic
├── convergence.go       # ConvergenceDetector
├── checkpoint.go        # Checkpointing system
├── evaluator.go         # ValidationEvaluator
└── *_test.go            # Comprehensive tests
```

### Core Data Structures

#### TrainingSample

```go
type TrainingSample struct {
    ID          string                    // Unique sample ID
    Query       string                    // Task/query
    Trajectory  *generator.Trajectory     // Execution trace
    Outcome     SampleOutcome             // success/failure
    GroundTruth *GroundTruth              // Optional expected output
    Metadata    map[string]string         // Additional context
}

type SampleOutcome string

const (
    OutcomeSuccess SampleOutcome = "success"
    OutcomeFailure SampleOutcome = "failure"
    OutcomePartial SampleOutcome = "partial"
)

type GroundTruth struct {
    ExpectedOutput string
    Correctness    float64 // 0.0 to 1.0
}
```

#### TrainingDataset

```go
type TrainingDataset struct {
    Samples       []*TrainingSample
    ValidationSet []*TrainingSample // Held-out for evaluation
    Metadata      DatasetMetadata
}

type DatasetMetadata struct {
    Name           string
    Description    string
    Domain         string
    TotalSamples   int
    ValidationSize int
    CreatedAt      time.Time
}

// Methods
func (d *TrainingDataset) Split(validationRatio float64) error
func (d *TrainingDataset) Shuffle(seed int64)
func (d *TrainingDataset) GetBatch(size int) []*TrainingSample
```

#### TrainingConfig

```go
type TrainingConfig struct {
    MaxEpochs          int       // Default: 5
    BatchSize          int       // Default: 10
    ConvergenceWindow  int       // Default: 2 (epochs to check)
    ConvergenceThreshold float64 // Default: 0.01 (1% improvement)
    CheckpointInterval int       // Default: 1 (every epoch)
    CheckpointDir      string    // Default: "./checkpoints"
    EarlyStopping      bool      // Default: true
    Seed               int64     // For reproducibility
}
```

#### EpochMetrics

```go
type EpochMetrics struct {
    EpochNumber       int
    TrainAccuracy     float64
    ValidationAccuracy float64
    PlaybookSize      int
    BulletsAdded      int
    BulletsRemoved    int
    AvgUtility        float64
    TrainingTime      time.Duration
}
```

### Components

#### 1. OfflineTrainer

**Purpose**: Main orchestrator for offline training process.

```go
type OfflineTrainer struct {
    playbook  *playbook.Playbook
    reflector *reflector.Reflector
    curator   curator.Curator
    evaluator *ValidationEvaluator
    detector  *ConvergenceDetector
    config    TrainingConfig
}

func NewOfflineTrainer(
    pb *playbook.Playbook,
    refl *reflector.Reflector,
    cur curator.Curator,
    config TrainingConfig,
) *OfflineTrainer

// Train trains the playbook on the dataset
func (t *OfflineTrainer) Train(ctx context.Context, dataset *TrainingDataset) (*TrainingResult, error)

// TrainEpoch executes one training epoch
func (t *OfflineTrainer) TrainEpoch(ctx context.Context, samples []*TrainingSample, epoch int) (*EpochMetrics, error)

// Checkpoint saves training state
func (t *OfflineTrainer) Checkpoint(epoch int) error

// Resume resumes training from checkpoint
func (t *OfflineTrainer) Resume(ctx context.Context, checkpointPath string) (*TrainingResult, error)
```

**Training Algorithm:**

```
For each epoch (1 to MaxEpochs):
  1. Shuffle training samples (if configured)
  2. For each batch of samples:
     a. Extract trajectories
     b. Reflect: trajectories → insights (Reflector)
     c. Curate: insights → bullets (Curator)
     d. Update playbook with new bullets
  3. Evaluate on validation set
  4. Record epoch metrics
  5. Check convergence
  6. Checkpoint playbook state
  7. If converged, early stop
```

#### 2. ValidationEvaluator

**Purpose**: Evaluate playbook performance on validation set.

```go
type ValidationEvaluator struct {
    playbook  *playbook.Playbook
    generator *generator.Generator
}

func NewValidationEvaluator(pb *playbook.Playbook, gen *generator.Generator) *ValidationEvaluator

// Evaluate runs validation on samples
func (e *ValidationEvaluator) Evaluate(ctx context.Context, samples []*TrainingSample) (*ValidationMetrics, error)

type ValidationMetrics struct {
    Accuracy       float64 // Correct predictions / total
    AvgConfidence  float64 // Average confidence score
    AvgLatency     time.Duration
    SuccessRate    float64 // Successful task completions
    FailureRate    float64
}
```

**Evaluation Process:**

1. For each validation sample:
   - Retrieve relevant bullets from playbook (top-K)
   - Generate prediction using bullets as context
   - Compare prediction with ground truth (if available)
   - Record success/failure and confidence
2. Aggregate metrics across all samples
3. Return ValidationMetrics

#### 3. ConvergenceDetector

**Purpose**: Detect when training has converged (no further improvement).

```go
type ConvergenceDetector struct {
    window    int     // Number of epochs to look back
    threshold float64 // Minimum improvement required
    history   []float64 // Validation accuracies
}

func NewConvergenceDetector(window int, threshold float64) *ConvergenceDetector

// HasConverged checks if training has plateaued
func (d *ConvergenceDetector) HasConverged(currentMetrics EpochMetrics) bool

// AddMetrics adds epoch metrics to history
func (d *ConvergenceDetector) AddMetrics(metrics EpochMetrics)

// GetTrend returns improvement trend (positive, negative, flat)
func (d *ConvergenceDetector) GetTrend() string
```

**Convergence Logic:**

```
Has converged if:
  (valAccuracy[epoch-N] - valAccuracy[epoch]) / valAccuracy[epoch-N] < threshold

Where N = convergence window (default: 2 epochs)
```

#### 4. Checkpointing System

**Purpose**: Save/restore training state for long runs.

```go
type Checkpoint struct {
    EpochNumber   int
    PlaybookState []byte // Serialized playbook
    Metrics       []EpochMetrics
    Timestamp     time.Time
}

func SaveCheckpoint(trainer *OfflineTrainer, epoch int, path string) error
func LoadCheckpoint(path string) (*Checkpoint, error)
func (t *OfflineTrainer) RestoreFromCheckpoint(checkpoint *Checkpoint) error
```

### Training Workflow

```
┌─────────────────────────────────────────────────────┐
│                 Offline Training                     │
└─────────────────────────────────────────────────────┘

1. Load Training Dataset
   ├── Training Samples (80%)
   └── Validation Samples (20%)

2. Initialize Playbook
   ├── Empty
   └── Or Pre-seeded

3. Training Loop (Epochs 1-5)
   │
   ├─ For Each Batch:
   │  ├── trajectories → Reflector → insights
   │  ├── insights → Curator → bullets
   │  └── bullets → Playbook (add/update)
   │
   ├─ Validation:
   │  ├── Run on validation set
   │  └── Compute accuracy, confidence
   │
   ├─ Convergence Check:
   │  ├── Compare with previous epochs
   │  └── Early stop if plateaued
   │
   └─ Checkpoint:
      └── Save playbook + metrics

4. Final Evaluation
   ├── Best epoch selection
   ├── Baseline comparison
   └── Export trained playbook
```

---

## API Reference

### OfflineTrainer

```go
// Create trainer
config := offline.TrainingConfig{
    MaxEpochs:          5,
    BatchSize:          10,
    ConvergenceWindow:  2,
    ConvergenceThreshold: 0.01,
    EarlyStopping:      true,
    CheckpointDir:      "./checkpoints",
    Seed:               42,
}
trainer := offline.NewOfflineTrainer(pb, refl, cur, config)

// Train on dataset
dataset := offline.LoadDataset("train.json")
result, err := trainer.Train(ctx, dataset)

// Result contains:
// - result.BestEpoch
// - result.Metrics (per epoch)
// - result.FinalPlaybook
// - result.BaselineComparison
```

### TrainingDataset

```go
// Create dataset from trajectories
samples := []*offline.TrainingSample{
    {
        ID:         "sample-1",
        Query:      "Write a function to parse JSON",
        Trajectory: traj1,
        Outcome:    offline.OutcomeSuccess,
    },
    // ... more samples
}
dataset := offline.NewDataset(samples)

// Split train/validation (80/20)
dataset.Split(0.2)

// Shuffle for randomness
dataset.Shuffle(42)

// Save/load
dataset.Save("dataset.json")
loaded, _ := offline.LoadDataset("dataset.json")
```

### ValidationEvaluator

```go
evaluator := offline.NewValidationEvaluator(pb, gen)
metrics, err := evaluator.Evaluate(ctx, validationSamples)

fmt.Printf("Accuracy: %.2f%%\n", metrics.Accuracy*100)
fmt.Printf("Success Rate: %.2f%%\n", metrics.SuccessRate*100)
```

### ConvergenceDetector

```go
detector := offline.NewConvergenceDetector(2, 0.01)

for epoch := 1; epoch <= 5; epoch++ {
    metrics := trainer.TrainEpoch(ctx, samples, epoch)
    detector.AddMetrics(metrics)
    
    if detector.HasConverged(metrics) {
        fmt.Printf("Converged at epoch %d\n", epoch)
        break
    }
}
```

---

## Performance Targets

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Single epoch (100 samples) | < 30s | Wall-clock time |
| Full training (1000 samples, 5 epochs) | < 1 hour | Wall-clock time |
| Validation (100 samples) | < 10s | Wall-clock time |
| Checkpoint save | < 1s | I/O time |
| Checkpoint load | < 1s | I/O time |
| Memory usage | < 2GB | Peak RSS |

---

## Testing Strategy

### Unit Tests

1. **TrainingSample Tests** (5 tests)
   - Create sample with all fields
   - Validate required fields
   - Serialize/deserialize

2. **TrainingDataset Tests** (8 tests)
   - Create dataset from samples
   - Split train/validation
   - Shuffle with seed (deterministic)
   - Get batches
   - Save/load JSON
   - Empty dataset handling
   - Large dataset (1000+ samples)

3. **OfflineTrainer Tests** (10 tests)
   - Single epoch training
   - Multi-epoch training
   - Playbook growth per epoch
   - Batch processing
   - Context cancellation
   - Error handling (invalid samples)

4. **ValidationEvaluator Tests** (6 tests)
   - Evaluate single sample
   - Evaluate batch
   - Accuracy calculation
   - Success rate calculation
   - Empty validation set
   - Missing ground truth handling

5. **ConvergenceDetector Tests** (7 tests)
   - Detect convergence (flat)
   - Detect improvement (not converged)
   - Detect regression
   - Window size behavior
   - Threshold sensitivity
   - Empty history

6. **Checkpoint Tests** (5 tests)
   - Save checkpoint
   - Load checkpoint
   - Resume training from checkpoint
   - Invalid checkpoint handling
   - Checkpoint directory creation

### Integration Tests

1. **Full Training Pipeline** (3 tests)
   - Train on synthetic dataset (50 samples)
   - Verify playbook growth
   - Verify validation metrics improve
   - Verify convergence detection
   - Verify checkpoint creation

2. **Resume Training** (2 tests)
   - Train for 2 epochs, checkpoint
   - Resume and train 3 more epochs
   - Verify continuity

3. **Baseline Comparison** (2 tests)
   - Train playbook vs empty baseline
   - Verify improvement in accuracy

### Coverage Target

- **Minimum**: 90% statement coverage
- **Critical paths**: 100% (training loop, evaluation, convergence)

---

## Deliverables

### Code
- `internal/ace/offline/` package (~800 lines)
- 40+ tests with 90%+ coverage
- Zero lint errors
- Race detector clean

### Documentation
- Updated `docs/packages/ace.md` with offline training section
- Training guide with examples
- Dataset format specification

### Examples
- Example training dataset (JSON)
- Example training configuration
- Example training script

---

## Dependencies

- **Feature 1**: Playbook for bullet storage
- **Feature 2**: Generator for trajectory creation
- **Feature 3**: Reflector for insight extraction
- **Feature 4**: Curator for bullet curation

---

## Risks and Mitigations

### Risk: Overfitting to Training Data

**Mitigation**: 
- Use train/validation split (80/20)
- Early stopping based on validation metrics
- Monitor validation vs training accuracy gap

### Risk: Long Training Times

**Mitigation**:
- Batch processing for efficiency
- Checkpointing for resumability
- Progress monitoring with ETA

### Risk: Poor Convergence

**Mitigation**:
- Configurable convergence window and threshold
- Manual epoch limit as fallback
- Detailed per-epoch metrics for debugging

### Risk: Dataset Quality Issues

**Mitigation**:
- Dataset validation on load
- Skip malformed samples with logging
- Require minimum dataset size

---

## Future Enhancements

1. **Hyperparameter Tuning**: Grid search over learning configurations
2. **Transfer Learning**: Fine-tune existing playbook on new domain
3. **Active Learning**: Select most informative samples for labeling
4. **Distributed Training**: Train on multiple machines for large datasets
5. **Advanced Metrics**: Precision, recall, F1 for specific bullet categories

---

## References

- ACE Paper: Section 4.3 (Offline Optimization)
- Feature 3 (Reflector): Insight extraction
- Feature 4 (Curator): Bullet curation
- Feature 6 (Grow-and-Refine): Playbook refinement

---

## Acceptance Criteria

1. ✅ Training loop executes multiple epochs
2. ✅ Validation metrics computed per epoch
3. ✅ Convergence detection stops training early
4. ✅ Checkpointing saves/resumes training state
5. ✅ Playbook grows with high-quality bullets
6. ✅ 90%+ test coverage
7. ✅ Zero lint errors
8. ✅ Race detector clean
9. ✅ Documentation updated
10. ✅ Example training dataset provided

---

**Status**: Ready for Implementation  
**Estimated Effort**: 3-4 hours (following micro-TDD workflow)
