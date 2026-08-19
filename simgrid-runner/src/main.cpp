#include <simgrid/s4u.hpp>

#include <algorithm>
#include <fstream>
#include <iostream>
#include <map>
#include <set>
#include <stdexcept>
#include <string>
#include <vector>

#include <nlohmann/json.hpp>

namespace sg4 = simgrid::s4u;
using json    = nlohmann::json;

struct Arguments {
  std::string platform;
  std::string input;
  std::string output;
};

struct TaskModel {
  std::string id;
  std::string assignment_id;
  std::string resource_id;
  double price_per_second = 0;
  double started_at       = 0;
  double finished_at      = 0;
  sg4::ExecPtr execution;
};

struct TransferModel {
  std::string producer_id;
  std::string consumer_id;
  std::string source_resource_id;
  std::string target_resource_id;
  long long bytes       = 0;
  double price_per_byte = 0;
  double started_at     = 0;
  double finished_at    = 0;
  sg4::CommPtr communication;
};

static Arguments parse_arguments(int argc, char** argv)
{
  Arguments arguments;
  for (int index = 1; index + 1 < argc; index += 2) {
    const std::string option = argv[index];
    if (option == "--platform")
      arguments.platform = argv[index + 1];
    else if (option == "--input")
      arguments.input = argv[index + 1];
    else if (option == "--output")
      arguments.output = argv[index + 1];
    else
      throw std::runtime_error("unknown argument: " + option);
  }
  if (arguments.platform.empty() || arguments.input.empty() || arguments.output.empty())
    throw std::runtime_error("usage: akoflow-simgrid-runner --platform FILE --input FILE --output FILE");
  return arguments;
}

static json read_json(const std::string& path)
{
  std::ifstream stream(path);
  if (!stream)
    throw std::runtime_error("cannot open input file: " + path);
  json document;
  stream >> document;
  return document;
}

static void write_json(const std::string& path, const json& document)
{
  std::ofstream stream(path);
  if (!stream)
    throw std::runtime_error("cannot open output file: " + path);
  stream << document.dump(2) << '\n';
}

static std::map<std::string, TaskModel> create_tasks(sg4::Engine& engine, const json& input)
{
  std::map<std::string, TaskModel> tasks;
  for (const auto& value : input.at("tasks")) {
    TaskModel task;
    task.id               = value.at("id").get<std::string>();
    task.assignment_id    = value.at("assignmentId").get<std::string>();
    task.resource_id      = value.at("resourceId").get<std::string>();
    task.price_per_second = value.value("pricePerSecond", 0.0);
    auto* host            = engine.host_by_name(task.resource_id);
    const double flops    = value.at("flops").get<double>();
    task.execution = sg4::Exec::init()->set_name(task.id)->set_flops_amount(std::max(0.0, flops))->set_host(host);
    if (!tasks.emplace(task.id, task).second)
      throw std::runtime_error("duplicate task: " + task.id);
  }
  return tasks;
}

static std::vector<TransferModel> connect_dependencies(const json& input, std::map<std::string, TaskModel>& tasks)
{
  std::vector<TransferModel> transfers;
  for (const auto& value : input.at("dependencies")) {
    const std::string producer_id = value.at("producerId").get<std::string>();
    const std::string consumer_id = value.at("consumerId").get<std::string>();
    auto& producer                = tasks.at(producer_id);
    auto& consumer                = tasks.at(consumer_id);
    const long long bytes         = value.value("bytes", 0LL);
    if (bytes <= 0 || producer.resource_id == consumer.resource_id) {
      producer.execution->add_successor(consumer.execution);
      continue;
    }
    TransferModel transfer;
    transfer.producer_id       = producer_id;
    transfer.consumer_id       = consumer_id;
    transfer.source_resource_id = producer.resource_id;
    transfer.target_resource_id = consumer.resource_id;
    transfer.bytes              = bytes;
    transfer.price_per_byte     = value.value("pricePerByte", 0.0);
    transfer.communication = sg4::Comm::sendto_init();
    producer.execution->add_successor(transfer.communication);
    transfer.communication->add_successor(consumer.execution);
    transfer.communication->set_name(producer_id + "->" + consumer_id)
        ->set_payload_size(static_cast<double>(bytes))
        ->set_source(producer.execution->get_host())
        ->set_destination(consumer.execution->get_host());
    transfers.push_back(transfer);
  }
  return transfers;
}

static void connect_lane_orders(const json& input, std::map<std::string, TaskModel>& tasks)
{
  for (const auto& value : input.at("resourceLaneOrders")) {
    const auto predecessor = value.at("predecessorId").get<std::string>();
    const auto successor   = value.at("successorId").get<std::string>();
    tasks.at(predecessor).execution->add_successor(tasks.at(successor).execution);
  }
}

static void run_simulation(sg4::Engine& engine, std::map<std::string, TaskModel>& tasks,
                           std::vector<TransferModel>& transfers)
{
  for (auto& [_, task] : tasks)
    task.execution->start();
  engine.run();
  for (auto& [_, task] : tasks) {
    task.started_at  = task.execution->get_start_time();
    task.finished_at = task.execution->get_finish_time();
  }
  for (auto& transfer : transfers) {
    transfer.started_at  = transfer.communication->get_start_time();
    transfer.finished_at = transfer.communication->get_finish_time();
  }
}

static std::map<std::string, double> task_costs(const std::map<std::string, TaskModel>& tasks,
                                                const std::map<std::string, double>& effective_starts)
{
  struct Window {
    double start  = 0;
    double finish = 0;
    double price  = 0;
    std::vector<std::string> task_ids;
  };
  std::map<std::string, Window> windows;
  for (const auto& [id, task] : tasks) {
    auto& window = windows[task.resource_id];
    const double start = effective_starts.at(id);
    const double finish = task.finished_at;
    if (window.task_ids.empty()) {
      window.start = start;
      window.finish = finish;
    } else {
      window.start = std::min(window.start, start);
      window.finish = std::max(window.finish, finish);
    }
    window.price = task.price_per_second;
    window.task_ids.push_back(id);
  }
  std::map<std::string, double> costs;
  for (const auto& [_, window] : windows) {
    const double total = std::max(0.0, window.finish - window.start) * window.price;
    double duration_sum = 0;
    for (const auto& id : window.task_ids)
      duration_sum += tasks.at(id).finished_at - effective_starts.at(id);
    for (const auto& id : window.task_ids) {
      const auto& task = tasks.at(id);
      const double duration = task.finished_at - effective_starts.at(id);
      costs[id] = total * (duration_sum > 0 ? duration / duration_sum : 1.0 / window.task_ids.size());
    }
  }
  return costs;
}

static json build_result(const json& input, const std::map<std::string, TaskModel>& tasks,
                         const std::vector<TransferModel>& transfers)
{
  json result = {
      {"runId", input.at("runId")}, {"planId", input.at("planId")}, {"mode", "simulation"},
      {"tasks", json::array()}, {"transfers", json::array()},
  };
  double makespan = 0;
  double compute = 0;
  double queue = 0;
  double transfer_seconds = 0;
  double total_cost = 0;
  std::map<std::string, double> data_ready;
  std::map<std::string, double> inbound_transfer;
  for (const auto& dependency : input.at("dependencies")) {
    const auto producer = dependency.at("producerId").get<std::string>();
    const auto consumer = dependency.at("consumerId").get<std::string>();
    data_ready[consumer] = std::max(data_ready[consumer], tasks.at(producer).finished_at);
  }
  for (const auto& transfer : transfers) {
    data_ready[transfer.consumer_id] = std::max(data_ready[transfer.consumer_id], transfer.finished_at);
    inbound_transfer[transfer.consumer_id] += std::max(0.0, transfer.finished_at - transfer.started_at);
  }
  auto execution_ready = data_ready;
  for (const auto& order : input.at("resourceLaneOrders")) {
    const auto predecessor = order.at("predecessorId").get<std::string>();
    const auto successor   = order.at("successorId").get<std::string>();
    execution_ready[successor] = std::max(execution_ready[successor], tasks.at(predecessor).finished_at);
  }
  // SimGrid fires Exec's start callback when an activity enters its started
  // lifecycle, including while it is vetoed by an unfinished predecessor.  The
  // execution trace exposed by AkôFlow must instead report when computation can
  // effectively begin, after all required data has arrived.
  std::map<std::string, double> effective_starts;
  for (const auto& [id, task] : tasks)
    effective_starts[id] = std::max(task.started_at, execution_ready[id]);
  const auto costs = task_costs(tasks, effective_starts);
  for (const auto& [id, task] : tasks) {
    const double start = effective_starts.at(id);
    const double finish = task.finished_at;
    const double duration = std::max(0.0, finish - start);
    makespan = std::max(makespan, finish);
    compute += duration;
    const double ready = data_ready[id];
    const double queued = std::max(0.0, start - ready);
    queue += queued;
    total_cost += costs.at(id);
    result["tasks"].push_back({
        {"id", input.at("runId").get<std::string>() + ":" + id},
        {"executionRunId", input.at("runId")}, {"planAssignmentId", task.assignment_id},
        {"activityId", id}, {"plannedResourceId", task.resource_id}, {"allocatedResourceId", task.resource_id},
        {"attempt", 1}, {"status", "completed"}, {"readyAt", ready}, {"dataReadyAt", ready},
        {"queuedAt", ready}, {"startedAt", start}, {"finishedAt", finish},
        {"runtimeSeconds", duration}, {"queueSeconds", queued}, {"transferSeconds", inbound_transfer[id]},
        {"interferenceSeconds", 0}, {"overheadSeconds", 0}, {"cost", costs.at(id)},
    });
  }
  for (const auto& transfer : transfers) {
    const double start = transfer.started_at;
    const double finish = transfer.finished_at;
    const double duration = std::max(0.0, finish - start);
    const double cost = static_cast<double>(transfer.bytes) * transfer.price_per_byte;
    transfer_seconds += duration;
    total_cost += cost;
    result["transfers"].push_back({
        {"id", input.at("runId").get<std::string>() + ":" + transfer.producer_id + ":" + transfer.consumer_id},
        {"executionRunId", input.at("runId")}, {"producerActivityId", transfer.producer_id},
        {"consumerActivityId", transfer.consumer_id}, {"sourceResourceId", transfer.source_resource_id},
        {"targetResourceId", transfer.target_resource_id}, {"bytes", transfer.bytes},
        {"startedAt", start}, {"finishedAt", finish}, {"durationSeconds", duration}, {"cost", cost},
    });
  }
  const double deadline = input.value("deadlineSeconds", 0.0);
  const double budget = input.value("budget", 0.0);
  const bool feasible = (deadline <= 0 || makespan <= deadline) && (budget <= 0 || total_cost <= budget);
  result["executed"] = {
      {"makespanSeconds", makespan}, {"cost", total_cost}, {"computeSeconds", compute},
      {"transferSeconds", transfer_seconds}, {"queueSeconds", queue}, {"interferenceSeconds", 0},
      {"overheadSeconds", 0}, {"feasible", feasible},
  };
  return result;
}

int main(int argc, char** argv)
{
  try {
    const Arguments arguments = parse_arguments(argc, argv);
    const json input           = read_json(arguments.input);
    sg4::Engine engine(&argc, argv);
    engine.load_platform(arguments.platform);
    auto tasks     = create_tasks(engine, input);
    auto transfers = connect_dependencies(input, tasks);
    connect_lane_orders(input, tasks);
    run_simulation(engine, tasks, transfers);
    write_json(arguments.output, build_result(input, tasks, transfers));
    return 0;
  } catch (const std::exception& error) {
    std::cerr << "akoflow-simgrid-runner: " << error.what() << '\n';
    return 1;
  }
}
