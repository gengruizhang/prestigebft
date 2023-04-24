"""
 Going through the computation engine does not mean this server will certainly win the ongoing campaign
 Here are several cases you may want to think harder:

 Suppose a server's current all past penalties are [1, 2, 3, 4, 5]

 By looking at the penalties, you can know the following information.
 1) this server was the leader of all past views
 2) this server is in view 5
 3) this server's current penalty is 5 in view 5
 4) there were 5 views have taken place so far

 NB: the # of views does not mean the values of views. Look at Eq.1, views can grow in different paces.

 During this view change, if
 1) this server wins in new_view
    a)
 2) this server looses in new_view
    a) it may become a worker in new_view
    b) it may

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