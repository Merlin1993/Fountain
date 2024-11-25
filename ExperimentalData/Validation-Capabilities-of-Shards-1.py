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

titleSize = 19
fontSize = 16
tickSize = 16
labelpad = 4


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

fileName = "D:\\go_workspace\\witCon\\core\\dataset\\Validation Capabilities of Shards"
shardNum = 32
dictoryName = f"Shard Counts\\{shardNum}"#"正式-分片-发送数量变化"
txSizeEx = [100000,110000,120000,130000] #[80000,90000,100000,110000,120000]
txSizeUinEx = [0,100000,110000,120000,130000] #[80000,90000,100000,110000,120000]


def filter_outliers(data, threshold=3):
    mean = np.mean(data)
    std_dev = np.std(data)

    # 计算数据范围
    lower_bound = mean - threshold * std_dev
    upper_bound = mean + threshold * std_dev

    # 过滤数据
    filtered_data = [x for x in data if lower_bound <= x <= upper_bound]
    return filtered_data


def filter_outliers_low(data, threshold=3):
    mean = np.mean(data)
    std_dev = np.std(data)

    # 计算数据范围
    lower_bound = mean - threshold * std_dev

    # 过滤数据
    filtered_data = [x for x in data if lower_bound <= x]
    return filtered_data


def filter_start_end(data):
    # 确保 data 至少有 3 个元素
    if len(data) < 3:
        return []

    # 返回除了第一个和最后一个元素的部分
    return data[0:13]


def filter_start_end_index(data,start,end):
    # 确保 data 至少有 3 个元素
    if len(data) < 3:
        return []

    # 返回除了第一个和最后一个元素的部分
    return data[start:end]


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


def sendSizeWithTotalTPSEvaluation():
    txSize = [110000]
    x_uniform = range(len(txSize))


    fig, ax1 = plt.subplots()
    for i in txSize:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\{dictoryName}\\send_{i}_node_0_pre_true_vs_true_shard_{shardNum}_signCore_28_shardCore_4_[]_2_aABeH54eGcQUgRbKq6MscXSdxGGuiWVHm.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)

        ax1.plot(range(len(TPS)), [int(n / 1000) for n in TPS], marker='o', color=c1,
             label=f"TPS_{i}")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax1.set_ylabel('TPS (K)', fontdict={'family': 'Times New Roman', 'size': 12})
    ax1.tick_params(axis='both', labelsize=10)
    ax1.grid(linestyle='--', axis="y")

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体
    ax1.set_xticks(x_uniform)
    ax1.set_xticklabels(txSize)

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': fontSize}

    # xy坐标轴
    plt.xlabel(r'shard size', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    for i in txSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\{dictoryName}\\send_{i}_node_1_pre_true_vs_true_shard_{shardNum}_'
        mfile = foundFile(file_part)
        tps_sum_array = None  #

        for f in mfile:
            TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)

            if tps_sum_array is None:
                tps_sum_array = np.array(TPS)  # 初始化为第一个文件的TPS数组
            else:
                if tps_sum_array.shape != TPS.shape:
                    # Pad the smaller array to match the larger one
                    max_len = max(tps_sum_array.shape[0], TPS.shape[0])
                    padded_tps_sum_array = np.pad(tps_sum_array, (0, max_len - tps_sum_array.shape[0]), mode='constant')
                    padded_TPS = np.pad(TPS, (0, max_len - TPS.shape[0]), mode='constant')

                    # Update tps_sum_array with the padded arrays
                    tps_sum_array = padded_tps_sum_array + padded_TPS
                else:
                    tps_sum_array += TPS
        tps_sum_array = filter_outliers(tps_sum_array,1)

        ax1.plot(range(len(tps_sum_array)), [int(n / 1000) for n in tps_sum_array], marker='*', color=c2, label=f'total_{i}')

    ax1.set_ylabel('count', fontdict={'family': 'Times New Roman', 'size': 12})
    ax1.tick_params(axis='both', labelsize=10)

    ax1.legend(loc="upper left", prop=legendFont)

    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t3_shard_TPS" + ".png", dpi=1024)
    plt.show()
    plt.close()


def sendSizeWithTPSEvaluation():
    sendTxSize = txSizeEx
    x_uniform = range(len(sendTxSize))

    TPS_values = []
    for i in sendTxSize:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\{dictoryName}\\send_{i}_node_0_pre_true_vs_true_shard_{shardNum}_signCore_28_shardCore_4_[]_2_aABeH54eGcQUgRbKq6MscXSdxGGuiWVHm.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)
        TPS_values.append(np.mean(filter_outliers(TPS, 3)))


    fig, ax1 = plt.subplots()
    ax1.plot(x_uniform, [int(n / 1000) for n in TPS_values], marker='o', color=c1,
             label=f"TPS_{i}")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax1.set_ylabel('TPS (K)', fontdict={'family': 'Times New Roman', 'size': 12})
    ax1.tick_params(axis='both', labelsize=10)
    ax1.grid(linestyle='--', axis="y")

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体
    ax1.set_xticks(x_uniform)
    ax1.set_xticklabels(sendTxSize)

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': fontSize}

    # xy坐标轴
    plt.xlabel(r'shard size', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    tps_mean_dict = {}
    for i in sendTxSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\{dictoryName}\\send_{i}_node_1_pre_true_vs_true_shard_{shardNum}_'
        mfile = foundFile(file_part)

        for f in mfile:
            TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)
            TPS_Mean = np.mean(filter_outliers(TPS, 3))

            match = re.search(r'\]_(.*)', f)
            suffix = match.group(1)
            if suffix not in tps_mean_dict or not tps_mean_dict[suffix]:
                # 如果suffix不存在或其值为空数组，初始化为含有TPS_Mean的数组
                tps_mean_dict[suffix] = [TPS_Mean]
            else:
                # 如果suffix存在并且数组非空，追加TPS_Mean
                tps_mean_dict[suffix].append(TPS_Mean)

    for k,v in tps_mean_dict.items():
        ax1.plot(range(len(v)), [int(n / 1000) for n in v], marker='*', color=c2, label=f'_nolegend_')

    max_tps = -float('inf')
    max_tps_key = None
    for k, v in tps_mean_dict.items():
        current_max = max(v)
        if current_max > max_tps:
            max_tps = current_max
            max_tps_key = k

    # 将找到的最大值标记在图表上
    if max_tps_key:
        x_max = np.argmax(tps_mean_dict[max_tps_key])  # 最大值在该数组中的索引
        ax1.axvline(x=x_max, color='gray', linestyle='--', linewidth=1)  # 添加垂直线
        ax1.text(
            x_max - 0.2,
            ax1.get_ylim()[0] + (ax1.get_ylim()[1] - ax1.get_ylim()[0]) * 0.02,
            f'Max Throughput: {int(max_tps):,} tx/s',
            color='red',
            ha='right',
            va='bottom',
            fontsize=14
        )

    ax1.set_ylabel('Count (K)', fontdict={'family': 'Times New Roman', 'size': 12})
    ax1.tick_params(axis='both', labelsize=10)

    ax1.legend(loc="upper left", prop=legendFont)

    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t3_send_size_TPS" + ".png", dpi=1024)
    plt.show()
    plt.close()


def sendSizeWithTPSSumEvaluation():

    sendTxSize = txSizeEx
    x_uniform = range(len(sendTxSize))

    TPS_values = []
    for i in sendTxSize:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\{dictoryName}\\send_{i}_node_0_pre_true_vs_true_shard_{shardNum}_signCore_28_shardCore_4_[]_2_aABeH54eGcQUgRbKq6MscXSdxGGuiWVHm.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)
        TPS_values.append(np.mean(filter_start_end_index(TPS,0,13)))


    fig, ax1 = plt.subplots(figsize=(8, 4))
    ax1.plot(x_uniform, [int(n / 1000) for n in TPS_values], marker='o', color=c1,
             label=f"Throughput of Core Peer")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax1.tick_params(axis='both', labelsize=10)
    ax1.grid(linestyle='--', axis="y")

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体
    ax1.set_xticks(x_uniform)
    ax1.set_xticklabels(sendTxSize)

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': fontSize}

    # xy坐标轴
    plt.xlabel(r'Transaction Sending Rate', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    tps_sum = []
    for i in sendTxSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\{dictoryName}\\send_{i}_node_1_pre_true_vs_true_shard_{shardNum}_'
        mfile = foundFile(file_part)
        sum = 0
        for f in mfile:
            TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)
            TPS = TPS[1:]
            TPS_Mean = np.mean(filter_start_end_index(TPS,0,13))

            # print(f,TPS_Mean)
            match = re.search(r'\]_(.*)', f)
            suffix = match.group(1)
            sum = sum + TPS_Mean
        tps_sum.append(sum)


    ax1.plot(range(len(sendTxSize)), [int(n / 1000) for n in tps_sum], marker='*', color=c2, label=f'Throughput Sum of Validation Peers')

    # 找到 tps_sum 的最大值
    max_tps_sum = max(tps_sum)

    # 在图表上添加水平线和注释
    ax1.axhline(y=max_tps_sum / 1000, color='red', linestyle='--', linewidth=1)  # 水平线
    ax1.text(
        len(sendTxSize) - 2,  # 注释位置，靠右显示
        max_tps_sum / 1000 + (ax1.get_ylim()[1] - ax1.get_ylim()[0]) * 0.02,  # 动态调整 y 位置
        f'Max Throughput: {int(max_tps_sum):,} tx/s',  # 注释文本
        color='red', ha='center', va='bottom', fontsize=12
    )

    ax1.set_ylabel('Throughput (Ktx/s)', fontdict={'family': 'Times New Roman', 'size': fontSize})
    ax1.tick_params(axis='both', labelsize=10)

    ax1.legend(loc="upper left", prop=legendFont)
    ax1.set_ylim(min(TPS_values) / 1000 * 0.9, max(TPS_values) / 1000 * 1.2)

    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t3_send_size_total_TPS" + ".png", dpi=1024)
    plt.show()
    plt.close()


# 3 验证节点CPU使用率箱型图的差异
def sendSizeWithCPUBoxEvaluation():
    sendTxSize = txSizeEx

    cpu_sum=[]
    for i in sendTxSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\{dictoryName}\\send_{i}_node_1_pre_true_vs_true_shard_{shardNum}_'
        mfile = foundFile(file_part)

        CPU_Vaules = []
        for f in mfile:
            TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)
            CPU_mean = np.mean(filter_outliers(CPU, 3))

            CPU_Vaules.append(CPU_mean)
        cpu_sum.append(CPU_Vaules)

    sendTxSize = txSizeUinEx
    x_uniform = range(len(sendTxSize))
    box = plt.boxplot(cpu_sum, vert=True, patch_artist=True,
                boxprops=dict(facecolor='lightblue'))  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接

    plt.xticks(x_uniform, sendTxSize)
    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体

    ax = plt.gca()

    # xy坐标轴
    plt.xlabel(r'Transaction Sending Rate', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("CPU Usage (%)", font, labelpad=labelpad)

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

    ax.tick_params(axis='both', labelsize=tickSize, pad=0)
    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t3_cpu_distribution" + ".png", dpi=1024)
    plt.show()
    plt.close()



# 3 验证节点CPU使用率箱型图的差异
def sendSizeWithNetBarEvaluation():
    sendTxSize = txSizeEx

    NetIn_sum=[]
    for i in sendTxSize:
        # 根据 i 动态生成文件名
        file_part = f'{fileName}\\{dictoryName}\\send_{i}_node_1_pre_true_vs_true_shard_{shardNum}_'
        mfile = foundFile(file_part)
        exclude_prefix = f'{fileName}\\{dictoryName}\\send_{i}_node_1_pre_true_vs_true_shard_{shardNum}_signCore_28_shardCore_4_[]'

        NetIn_Vaules = []
        for f in mfile:
            if not f.startswith(exclude_prefix):
                TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(f)
                NetIn_mean = np.mean(filter_outliers(NetIn, 3))

                NetIn_Vaules.append(NetIn_mean/1000/1000)
        NetIn_sum.append(NetIn_Vaules)

    sendTxSize = txSizeUinEx
    x_uniform = range(len(sendTxSize))
    box = plt.boxplot(NetIn_sum, vert=True, patch_artist=True,
                boxprops=dict(facecolor='lightblue'))  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接

    plt.xticks(x_uniform, sendTxSize)
    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体

    ax = plt.gca()

    # xy坐标轴
    plt.xlabel(r'Transaction Sending Rate', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("Network Incoming Traffic (MB)", font, labelpad=labelpad)

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

    ax.tick_params(axis='both', labelsize=tickSize, pad=0)
    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t3_net_distribution" + ".png", dpi=1024)
    plt.show()
    plt.close()


if __name__ == '__main__':

    sendSizeWithTotalTPSEvaluation()

    sendSizeWithTPSEvaluation()

    sendSizeWithCPUBoxEvaluation()

    sendSizeWithNetBarEvaluation()

    sendSizeWithTPSSumEvaluation()