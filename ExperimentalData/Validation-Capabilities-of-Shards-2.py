import glob
import re

import pandas as pd
import numpy as np
from matplotlib.pyplot import xticks
from scipy import stats
from matplotlib import pyplot as plt
from matplotlib import rcParams

config = {
    "mathtext.fontset": 'stix',
}
rcParams.update(config)

titleSize = 16
fontSize = 16
tickSize = 16
labelpad = 4

bigTitleSize = 19
bigFontSize = 19
bigTickSize = 19


#
# c1 = "orange"
# c2 = "firebrick"
# c3 = "cyan"

def toColor(a, b, c):
    return a / 256, b / 256, c / 256


c1 = toColor(255, 125, 25)
c2 = toColor(149, 27, 27)
c3 = toColor(130, 220, 233)
c4 = toColor(130, 220, 27)

# 各种变量
init_title = 0
TPS_data = init_title + 1
Latency_data = init_title + 2
BPS_data = init_title + 3
CPU_data = init_title + 4
NetIn_data = init_title + 5
NetOut_data = init_title + 6

shardSizeEx = [16, 32, 64, 128]
shardSizeUinEx = [0, 16, 32, 64, 128]

fileName = "D:\\go_workspace\\witCon\\core\\dataset\\Validation Capabilities of Shards"


def filter_outliers(data, threshold=3):
    mean = np.mean(data)
    std_dev = np.std(data)

    # 计算下限，忽略上限
    lower_bound = mean - threshold * std_dev

    # 只过滤掉低于下限的值，保留其他值
    filtered_data = [x for x in data if x >= lower_bound]

    return filtered_data


def filter_start_end(data):
    if len(data) < 3:
        return []
    return data[1:-1]


def readExecl(file):
    df = pd.read_excel(file, sheet_name="4")
    TPS = df.iloc[TPS_data, 1:]
    Latency = df.iloc[Latency_data, 1:]
    BPS = df.iloc[BPS_data, 1:]
    CPU = df.iloc[CPU_data, 1:]
    NetIn = df.iloc[NetIn_data, 1:]
    NetOut = df.iloc[NetOut_data, 1:]
    return TPS, Latency, BPS, CPU, NetIn, NetOut


def foundFile(fixed_part):
    file_pattern = fixed_part + "*"
    matching_files = glob.glob(file_pattern)
    return matching_files


def sendSizeWithTPSEvaluation():
    shardSize = shardSizeEx
    x_uniform = range(len(shardSize))

    TPS_values = []
    for i in shardSize:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\正式-分片-分片数量变化\\{i}\\send_100000_node_0_pre_true_vs_true_shard_{i}_signCore_28_shardCore_4_[]_2_aABeH54eGcQUgRbKq6MscXSdxGGuiWVHm.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)
        TPS_values.append(np.mean(filter_start_end(TPS)))

    fig, ax1 = plt.subplots()
    ax1.plot(x_uniform, [int(n / 1000) for n in TPS_values], marker='o', color=c1,
             label=f"Throughput of Core Peer")
    ax1.set_ylabel('Throughput (Ktx/s)', fontdict={'family': 'Times New Roman', 'size': bigFontSize})
    ax1.tick_params(axis='both', labelsize=14)
    ax1.grid(linestyle='--', axis="y")

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigFontSize}  # 标题的字体
    ax1.set_xticks(x_uniform)
    ax1.set_xticklabels(shardSize)

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': bigFontSize}

    # xy坐标轴
    plt.xlabel(r'Number of Shards', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    tps_total = []
    for i in shardSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\正式-分片-分片数量变化\\{i}\\send_100000_node_1_pre_true_vs_true_shard_{i}_'
        mfile = foundFile(file_part)
        total = 0
        for f in mfile:
            TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)
            TPS_Mean = np.mean(filter_start_end(TPS))
            total += TPS_Mean
        tps_total.append(total / 1000)

        print( tps_total)

    ax1.plot(x_uniform, tps_total, marker='*', color=c2, label=f'Throughput Sum of Validation Peers')

    ax1.set_ylabel('Throughput (Ktx/s)', fontdict={'family': 'Times New Roman', 'size': bigFontSize})
    ax1.tick_params(axis='both', labelsize=14)
    ax1.set_ylim(min(TPS_values) / 1000 * 0.9, max(TPS_values) / 1000 * 1.2)

    ax1.legend(loc="upper left", prop=legendFont)

    plt.subplots_adjust(bottom=0.13)
    plt.savefig("p2_t4_shard_send_size_TPS" + ".png", dpi=1024)
    plt.show()
    plt.close()


def sendSizeWithCPUBoxEvaluation():
    shardSize = shardSizeEx

    cpu_sum = []
    for i in shardSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\正式-分片-分片数量变化\\{i}\\send_100000_node_1_pre_true_vs_true_shard_{i}'
        mfile = foundFile(file_part)

        CPU_Vaules = []
        for f in mfile:
            TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)
            CPU_mean = np.mean(filter_outliers(CPU, 3))

            CPU_Vaules.append(CPU_mean)
        cpu_sum.append(CPU_Vaules)

    sendTxSize = shardSizeUinEx
    x_uniform = range(len(sendTxSize))
    box = plt.boxplot(cpu_sum, vert=True, patch_artist=True,
                      boxprops=dict(facecolor='lightblue'))  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    for boxes in box['boxes']:
        boxes.set_color(c1)

    # 其他元素的颜色也可以自定义，比如中位线、胡须线等
    for median in box['medians']:
        median.set_color("black")  # 设置中位线颜色

    for whisker in box['whiskers']:
        whisker.set_color("black")  # 设置胡须线颜色

    for cap in box['caps']:
        cap.set_color('gray')  # 设置盒子上限和下限颜色

    for flier in box['fliers']:
        flier.set(marker='o', color='red', alpha=0.5, markersize=4)

    plt.xticks(x_uniform, sendTxSize)
    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigTitleSize}  # 标题的字体

    ax = plt.gca()

    # xy坐标轴
    plt.xlabel(r'Number of Shards', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("CPU Usage (%)", font, labelpad=labelpad)

    ax.tick_params(axis='both', labelsize=bigTickSize, pad=0)
    plt.subplots_adjust(bottom=0.13,left=0.16)
    plt.savefig("p2_t4_shard_cpu_distribution" + ".png", dpi=1024)
    plt.show()
    plt.close()


# 3 验证节点CPU使用率箱型图的差异
def sendSizeWithShardNetBarEvaluation():
    shardSize = shardSizeEx

    NetIn_sum = []
    for i in shardSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\正式-分片-分片数量变化\\{i}\\send_100000_node_1_pre_true_vs_true_shard_{i}'
        mfile = foundFile(file_part)
        exclude_prefix = f'{fileName}\\正式-分片-分片数量变化\\{i}\\send_100000_node_1_pre_true_vs_true_shard_{i}_signCore_28_shardCore_4_[]'

        NetIn_Vaules = []
        for f in mfile:
            if not f.startswith(exclude_prefix):
                TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)
                NetIn_mean = np.mean(filter_outliers(NetIn, 3))

                NetIn_Vaules.append(NetIn_mean / 1000 / 1000)
        NetIn_sum.append(NetIn_Vaules)

    sendTxSize = shardSizeUinEx
    x_uniform = range(len(sendTxSize))
    box = plt.boxplot(NetIn_sum, vert=True, patch_artist=True,
                      boxprops=dict(facecolor='lightblue'))  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    for boxes in box['boxes']:
        boxes.set_color(c1)

    # 其他元素的颜色也可以自定义，比如中位线、胡须线等
    for median in box['medians']:
        median.set_color("black")  # 设置中位线颜色

    for whisker in box['whiskers']:
        whisker.set_color("black")  # 设置胡须线颜色

    for cap in box['caps']:
        cap.set_color('gray')  # 设置盒子上限和下限颜色

    for flier in box['fliers']:
        flier.set(marker='o', color='red', alpha=0.5, markersize=4)

    plt.xticks(x_uniform, sendTxSize)
    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigTitleSize}  # 标题的字体

    ax = plt.gca()

    # xy坐标轴
    plt.xlabel(r'Number of Shards', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("Network Incoming Traffic (MB)", font, labelpad=labelpad)

    ax.tick_params(axis='both', labelsize=bigTickSize, pad=0)
    plt.subplots_adjust(bottom=0.13)
    plt.savefig("p2_t4_shard_net_distribution" + ".png", dpi=1024)
    plt.show()
    plt.close()


# ok 3 共识节点的CPU和网络输出的差异
def sendSizeWithNetBarEvaluation():
    shardSize = shardSizeEx

    NetOut_sum = []
    CPU_data = []
    for i in shardSize:
        # 根据 i 动态生成文件名
        _file = f'{fileName}\\正式-分片-分片数量变化\\{i}\\send_100000_node_0_pre_true_vs_true_shard_{i}_signCore_28_shardCore_4_[]_2_aABeH54eGcQUgRbKq6MscXSdxGGuiWVHm.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(_file)
        NetOut_mean = np.mean(filter_outliers(NetOut, 3))
        CPU_mean = np.mean(filter_outliers(CPU, 3))  # 添加 CPU 的计算
        NetOut_sum.append(NetOut_mean / 1000 / 1000)
        CPU_data.append(CPU_mean)  # 填充 CPU 数据

    x_uniform = range(len(shardSize))
    fig, ax1 = plt.subplots()


    ax1.bar(x_uniform, CPU_data, width=0.4, color=c1, label=r"CPU Usage")
    ax1.tick_params(axis='both', labelsize=14)


    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigFontSize}
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': bigFontSize}
    #

    ax1.set_ylim(0, max(CPU_data) * 1.2)

    # 设置 x 和 y 轴的标签
    plt.xlabel('Number of Shards', font, labelpad=labelpad)
    ax1.set_ylabel("CPU Usage (%)", font, labelpad=labelpad)

    ax2 = ax1.twinx()
    ax2.plot(x_uniform, NetOut_sum, marker='o', color=c2,
             label=r"Network Outgoing Traffic")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax2.tick_params(axis='both', labelsize=14)
    ax2.grid(linestyle='--', axis="y")
    ax2.set_ylim(0, max(NetOut_sum) * 1.4)  # 增加 y 轴顶部留白
    ax2.set_ylabel("Network Outgoing Traffic (MB)", font, labelpad=labelpad)


    # # 设置图例分别显示
    handles1, labels1 = ax1.get_legend_handles_labels()
    handles2, labels2 = ax2.get_legend_handles_labels()
    # 合并图例
    ax1.legend(handles1 + handles2, labels1 + labels2, loc="upper left", prop=legendFont)

    ax1.set_xticks(x_uniform)
    print(shardSize)
    print(x_uniform)
    ax1.set_xticklabels(shardSize)

    plt.subplots_adjust(right=0.88, bottom=0.13)
    plt.savefig("p2_t4_consensus_node_netcpu" + ".png", dpi=1024)
    plt.show()
    plt.close()


if __name__ == '__main__':
    # 四张图
    # 查看验证节点和共识节点的TPS是否相当，共识节点的性能是否相当。
    sendSizeWithNetBarEvaluation()
    sendSizeWithTPSEvaluation()
    # # #
    # # # # #分片节点的性能参数
    sendSizeWithCPUBoxEvaluation()
    sendSizeWithShardNetBarEvaluation()
