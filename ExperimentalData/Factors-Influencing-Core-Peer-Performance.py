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
fontSize = 19
tickSize = 19
bigLabelsize = 14
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

fileName = "D:\\go_workspace\\witCon\\core\\dataset\\Factors Influencing Core Peer Performance"


def filter_outliers(data, threshold=4):
    sorted_data = sorted(data)

    # 排除最小的 num_smallest 个值
    filtered_data = sorted_data[threshold:len(sorted_data)-threshold]

    return filtered_data


def filter_outliersDown(data, threshold=4):
    # 将数据按升序排序
    sorted_data = sorted(data)

    # 排除最小的 num_smallest 个值
    filtered_data = sorted_data[threshold:]

    return filtered_data

def readExecl(file):
    df = pd.read_excel(file, sheet_name="4")
    TPS = df.iloc[TPS_data, 1:]
    Latency = df.iloc[Latency_data, 1:10]
    BPS = df.iloc[BPS_data, 1:]
    CPU = df.iloc[CPU_data, 1:]
    NetIn = df.iloc[NetIn_data, 1:]
    NetOut = df.iloc[NetOut_data, 1:]
    return TPS, Latency, BPS, CPU, NetIn, NetOut


def shardSizeWithTPSEvaluation():
    shard_size = [4, 8, 16, 32, 64, 128, 256, 512]
    x_uniform = range(len(shard_size))

    TPS_values = []
    Latency_values = []
    CPU_values = []
    NetIn_values = []
    NetOut_values = []

    for i in shard_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\1.Number of Shards\\send_100000_node_0_pre_true_vs_true_shard_{i}_signCore_28_shardCore_4_[]_3_QVda23jgsjbQ2d3MSF3iK1aVrCiwGbVmq.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)

        TPS_values.append(np.mean(filter_outliers(TPS, 1)))
        Latency_values.append(np.mean(filter_outliers(Latency, 3)))
        CPU_values.append(np.mean(filter_outliers(CPU, 3)))
        NetIn_values.append(np.mean(filter_outliers(NetIn, 3)))
        NetOut_values.append(np.mean(filter_outliers(NetOut, 3)))

    fig, ax1 = plt.subplots()
    ax1.plot(x_uniform, [int(n / 1000) for n in TPS_values], marker='o', color=c1,
             label=r"Throughput")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax1.set_ylabel('Throughput (Ktx/s)', fontdict={'family': 'Times New Roman', 'size': fontSize})
    ax1.tick_params(axis='both', labelsize=bigLabelsize)
    ax1.grid(linestyle='--', axis="y")

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体
    ax1.set_xticks(x_uniform)
    ax1.set_xticklabels(shard_size)
    ax1.set_ylim(0, max(TPS_values) /1000 * 1.2)

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': fontSize}

    # xy坐标轴
    plt.xlabel(r'Number of Shards', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    ax2 = ax1.twinx()
    ax2.plot(x_uniform, [int(n / 1000) for n in Latency_values], marker='*', color=c2, label="Latency")
    ax2.set_ylabel('Latency (ms)', fontdict={'family': 'Times New Roman', 'size': fontSize})
    ax2.tick_params(axis='both', labelsize=bigLabelsize)
    ax2.set_ylim(0, max(Latency_values)/1000 * 1.2)

    ax1.legend(loc="upper left", prop=legendFont)
    ax2.legend(loc="upper right", prop=legendFont)

    plt.tight_layout(pad=1.0)
    plt.subplots_adjust(bottom=0.13, right=0.85)
    plt.savefig("t1_shard_TPS_latency" + ".png", dpi=1024)
    plt.show()
    plt.close()


def cpuCoreWithTPSEvaluation():
    cpu_size = [20, 22, 24, 26, 28, 30]
    x_uniform = range(len(cpu_size))

    TPS_values = []
    Latency_values = []
    CPU_values = []
    NetIn_values = []
    NetOut_values = []
    P2_CPU_values = []

    for i in cpu_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\2.CPU Cores\\send_100000_node_0_pre_true_vs_true_shard_32_signCore_{i}_shardCore_4_[]_0_d6tqXy2Zv1Boz5xH4oLhmgjizFkHzBAsL.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)

        TPS_values.append(np.mean(filter_outliersDown(TPS, 2)))
        Latency_values.append(Latency[len(Latency)-1])
        CPU_values.append(int(np.mean(filter_outliers(CPU, 1))))
        NetIn_values.append(np.mean(filter_outliers(NetIn, 3)))
        NetOut_values.append(np.mean(filter_outliers(NetOut, 3)))

    for i in cpu_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\正式2-核数变化\\send_100000_node_1_pre_true_vs_true_shard_32_signCore_{i}_shardCore_4_[]_2_ds7gA3Bf6VhLAv2E1rBALbxKryF35Ttme.xlsx'
        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)
        P2_CPU_values.append(int(np.mean(filter_outliers(CPU, 3))))

    fig, ax1 = plt.subplots()
    ax1.plot(x_uniform, [int(n / 1000) for n in TPS_values], marker='o', color=c1,
             label=r"Throughput")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax1.set_ylabel('Throughput (Ktx/s)', fontdict={'family': 'Times New Roman', 'size': fontSize})
    ax1.tick_params(axis='both', labelsize=bigLabelsize)
    ax1.grid(linestyle='--', axis="y")

    bars = ax1.bar(x_uniform, CPU_values, width=0.4, color=c1, label=r"CPU Usage")

    ax1.set_ylim(0, max(TPS_values)/1000 * 1.5)

    # 在柱状图顶部显示数值
    for bar in bars:
        yval = bar.get_height()  # 获取每个柱子的高度
        ax1.text(bar.get_x() + bar.get_width() / 2, yval, f'{yval}%', ha='center', va='bottom', fontsize=bigLabelsize)

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体
    ax1.set_xticks(x_uniform)
    ax1.set_xticklabels(cpu_size)

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': fontSize}

    # xy坐标轴
    plt.xlabel(r'Number of CPU Cores', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    ax2 = ax1.twinx()
    ax2.plot(x_uniform, [int(n / 1000) for n in Latency_values], marker='*', color=c2, label="Latency")
    ax2.set_ylabel('Latency (ms)', fontdict={'family': 'Times New Roman', 'size': fontSize})
    ax2.tick_params(axis='both', labelsize=bigLabelsize)

    ax2.set_ylim(0, max(Latency_values)/1000 * 1.4)

    ax1.legend(loc="upper left", prop=legendFont)
    ax2.legend(loc="upper right", prop=legendFont)
    plt.tight_layout(pad=1.0)

    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t1_cpu_TPS" + ".png", dpi=1024)
    plt.show()
    plt.close()


# 交易打包大小和TPS的关系
def packSizeWithTPSEvaluation():
    pack_size = [2, 5, 10, 15,20, 25,30]
    # x_uniform = range(len(cpu_size))

    TPS_values = []
    Latency_values = []
    CPU_values = []
    NetIn_values = []
    NetOut_values = []

    for i in pack_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\3.Maximum Number of Transactions per Block\\{i}\\send_100000_node_0_pre_true_vs_true_shard_32_signCore_28_shardCore_4_[]_0_d6tqXy2Zv1Boz5xH4oLhmgjizFkHzBAsL.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)

        TPS_values.append(np.mean(filter_outliersDown(TPS, 3)))
        Latency_values.append(np.mean(filter_outliers(Latency, 3)))
        CPU_values.append(np.mean(filter_outliers(CPU, 3)))
        NetIn_values.append(np.mean(filter_outliers(NetIn, 3)))
        NetOut_values.append(np.mean(filter_outliers(NetOut, 3)))

    fig, ax1 = plt.subplots()
    ax1.plot([n*1000 for n in pack_size], [int(n / 1000) for n in TPS_values], marker='o', color=c1)  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax1.set_ylabel('Throughput (Ktx/s)', fontdict={'family': 'Times New Roman', 'size': fontSize})
    ax1.tick_params(axis='both', labelsize=bigLabelsize)
    ax1.grid(linestyle='--', axis="y")

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': fontSize}

    # xy坐标轴
    plt.xlabel(r'Transaction Amount per Block', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    plt.tight_layout(pad=1.0)
    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t1_pack_size_TPS" + ".png", dpi=1024)
    plt.show()
    plt.close()


def sendSizeLShardEvaluation(shard_size):
    send_size = [70000,80000, 90000, 100000, 110000, 120000, 130000, 140000]

    TPS_values = []
    Latency_values = []
    CPU_values = []
    NetIn_values = []
    NetOut_values = []

    for i in send_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\4.Transaction Send Rate\\send_{i}_node_0_pre_true_vs_true_shard_{shard_size}_signCore_28_shardCore_4_[]_0_d6tqXy2Zv1Boz5xH4oLhmgjizFkHzBAsL.xlsx'

        TPS, Latency, BPS, CPU, NetIn, NetOut = readExecl(file_name)

        TPS = TPS[3:]

        TPS_values.append(np.mean(filter_outliersDown(TPS, 1)))
        Latency_values.append(Latency[len(Latency)-1])
        CPU_values.append(int(np.mean(filter_outliers(CPU, 3))))
        NetIn_values.append(np.mean(filter_outliers(NetIn, 3)))
        NetOut_values.append(np.mean(filter_outliers(NetOut, 3)))

    fig, ax1 = plt.subplots()
    plots = ax1.plot([n / 1000 for n in TPS_values], [int(n / 1000) for n in Latency_values], marker='o', color=c1)  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    ax1.set_ylabel('Latency (ms)', fontdict={'family': 'Times New Roman', 'size': fontSize})
    ax1.tick_params(axis='both', labelsize=bigLabelsize)
    ax1.grid(linestyle='--', axis="y")

    # 获取 X 轴的最大值
    x_max = max(TPS_values)

    # 在 X 轴最大值处添加虚线标识
    plt.axvline(x=x_max/1000, color='gray', linestyle='--', linewidth=1)
    plt.text(x_max/1000 - 0.2, ax1.get_ylim()[0] + (ax1.get_ylim()[1] - ax1.get_ylim()[0]) * 0.02,
             f'Max Throughput: {int(x_max):,} tx/s', color='red', ha='right', va='bottom', fontsize=14)

    i = 0
    for i, (x, y) in enumerate(zip([n / 1000 for n in TPS_values], [int(n / 1000) for n in Latency_values])):
        ax1.text(x, y, f'{int(send_size[i]/1000)}', ha='center', va='bottom', fontsize=14)

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}  # 标题的字体

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': fontSize}

    # xy坐标轴
    plt.xlabel(r'Throughput (Ktx/s)', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")

    plt.tight_layout(pad=1.0)
    plt.subplots_adjust(bottom=0.13)
    plt.savefig("t1_TPS_latency" + ".png", dpi=1024)
    plt.show()
    plt.close()


if __name__ == '__main__':
    shardSizeWithTPSEvaluation()
    cpuCoreWithTPSEvaluation()
    packSizeWithTPSEvaluation()
    sendSizeLShardEvaluation(32)
