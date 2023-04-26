"""
"""
import statistics
import math


def penalization(cur_view, new_view, cur_penalty):
    new_penalty = (new_view - cur_view) * cur_penalty + 1
    # print("Eq1: penalty after penalization is: ", new_penalty)
    return new_penalty


def compensation(list_of_all_penalties, new_penalty, t_left, t_all):
    latest_penalty = list_of_all_penalties[-1]
    mu = statistics.mean(list_of_all_penalties)
    sigma = statistics.pstdev(list_of_all_penalties)
    deduction = math.floor(t_left / t_all * (latest_penalty - mu) / sigma)
    new_penalty = new_penalty - max(0, deduction)

    # print("Eq2: penalty after compensation is: ", new_penalty)
    return new_penalty

# compensation([1, 2, 3, 4, 5], penalization(1, 2, 5), 1, 1)

def calculator(list_of_all_penalties, cur_view, new_view, t_left, t_all):
    new_penalty = penalization(cur_view, new_view, list_of_all_penalties[-1])
    return compensation(list_of_all_penalties, new_penalty, t_left, t_all)