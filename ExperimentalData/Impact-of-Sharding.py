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

titleSize = 14
bigTitleSize = 19
bigFontSize = 19
bigTickSize = 19
MiddleTitleSize = 16
MiddleFontSize = 16
MiddleTickSize = 16
fontSize = 14
tickSize = 14
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
heightX = 0
shardSizeX = 1
txTotalSizeX = 2
shardTotalSizeX = 3
executeTotalTimeX = 6
executeTxTimeX = 7
proofTimex = 8
verifyTimex = 9


def shardTxOffsetX(i):
    return verifyTimex + 4 * i + 1


def shardSizeOffsetX(i):
    return verifyTimex + 4 * i + 2


def shardToMeOffsetX(i):
    return verifyTimex + 4 * i + 3


def shardToOtherOffsetX(i):
    return verifyTimex + 4 * i + 4


fileName = "D:\\go_workspace\\witCon\\core\\dataset\\Impact of Sharding"

def shardSizeWithNetEvaluationAppendShard(tx_size):
    shard_size = [4, 8, 16, 32, 64, 128, 256]
    x_uniform = range(len(shard_size))

    average_values = []
    average_tx_size_values = []
    average_1_values = []
    average_2_values = []

    for i in shard_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\ShardExprimentData_{i}_{tx_size}_0.xlsx'

        # 读取 Excel 文件中的指定 sheet
        df = pd.read_excel(file_name, sheet_name="shard")

        # 从第2行开始读取 B 列数据（注意 DataFrame 的索引是从 0 开始，所以需要从 index 1 开始读取）
        b_column = df.iloc[1:, 1]

        # 计算 B 列的平均数
        average = b_column.mean()
        average_values.append(int(average))

        b2_column = df.iloc[1:, 2]

        # 计算 B 列的平均数
        average_tx_size = b2_column.mean()
        average_tx_size_values.append(int(average_tx_size))


    for i in shard_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\ShardExprimentData_{i}_{tx_size}_2.xlsx'

        # 读取 Excel 文件中的指定 sheet
        df = pd.read_excel(file_name, sheet_name="shard")

        # 从第2行开始读取 B 列数据（注意 DataFrame 的索引是从 0 开始，所以需要从 index 1 开始读取）
        b1_column = df.iloc[1:, 1]
        average_size = b1_column.mean()
        # 增加分片的位图
        average_size = average_size + i*i*i/8
        average_2_values.append(int(average_size))

    # 将平均数列表转换为 NumPy 数组
    xdata = np.array(average_values)
    txdata = np.array(average_tx_size_values)
    txdata1 = np.array(average_1_values)
    txdata2 = np.array(average_2_values)
    print(xdata)
    plt.plot(x_uniform, [(n / 1024 /1024) for n in txdata], marker='o', color=c1,
             label=r"Baseline")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    plt.plot(x_uniform, [(n / 1024/1024) for n in xdata], marker='*', color=c2,
             label=r"No Optimization")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    # plt.plot(x_uniform, [(n / 1024/1024) for n in txdata1], marker='*', color=c3,
    #          label=r"One-Stage Optimization")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    plt.plot(x_uniform, [(n / 1024/1024) for n in txdata2], marker='*', color=c4,
             label=r"Optimization")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    plt.fill_between(
        x_uniform, [(n / 1024/1024) for n in txdata], color=c1,
        alpha=0.3  # 填充颜色的透明度
    )
    plt.fill_between(
        x_uniform, [(n / 1024/1024) for n in xdata], color=c2,
        alpha=0.3  # 填充颜色的透明度
    )

    plt.fill_between(
        x_uniform, [(n / 1024/1024) for n in txdata2], color=c4,
        alpha=0.3  # 填充颜色的透明度
    )

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigTitleSize}  # 标题的字体
    plt.xticks(x_uniform, shard_size)
    # plt.title(title, font)  # 标题
    ax = plt.gca()


    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': bigFontSize}
    ax.legend(loc="best", prop=legendFont)

    # xy坐标轴
    plt.xlabel(r'Number of Shards', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("Data Size (MB)", font, labelpad=labelpad)

    ax2 = ax.twinx()
    tx_base_value = txdata[0]/1024/1024
    ax2.set_ylim(ax.get_ylim()/tx_base_value)
    # ax2.plot(x_uniform, tx_ratios, color='blue', linestyle='--', label='Ratio to tx')

    # 右侧y轴标签
    ax2.set_ylabel('Ratio to Baseline', fontdict={'family': 'Times New Roman', 'size': bigTitleSize})

    ax.tick_params(axis='both', labelsize=bigTickSize, pad=0)
    ax2.tick_params(axis='both', labelsize=bigTickSize, pad=0)
    # ax.set_xlabel('x label', fontsize=fontSize, labelpad=labelpad)
    # ax.set_ylabel('y label', fontsize=fontSize, labelpad=labelpad)
    plt.subplots_adjust(bottom=0.13)
    plt.savefig(f"t2_shard_bitmap_{tx_size}" + ".png", dpi=1024)
    # plt.show()
    plt.close()


def shardSizeWithTimeEvaluation(tx_size):
    shard_size = [4, 8, 16, 32, 64, 128, 256]
    x_uniform = range(len(shard_size))

    average_values = []
    average_tx_size_values = []
    average_1_values = []
    average_2_values = []

    for i in shard_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\ShardExprimentData_{i}_{tx_size}_0.xlsx'

        # 读取 Excel 文件中的指定 sheet
        df = pd.read_excel(file_name, sheet_name="shard")

        # 从第2行开始读取 B 列数据（注意 DataFrame 的索引是从 0 开始，所以需要从 index 1 开始读取）
        b_column = df.iloc[1:, 6]

        # 计算 B 列的平均数
        average = b_column.mean()
        average_values.append(int(average))

        b2_column = df.iloc[1:, 5]

        # 计算 B 列的平均数
        average_tx_size = b2_column.mean()
        average_tx_size_values.append(int(average_tx_size))

    for i in shard_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\ShardExprimentData_{i}_{tx_size}_2.xlsx'

        # 读取 Excel 文件中的指定 sheet
        df = pd.read_excel(file_name, sheet_name="shard")

        # 从第2行开始读取 B 列数据（注意 DataFrame 的索引是从 0 开始，所以需要从 index 1 开始读取）
        b1_column = df.iloc[1:, 5]
        average_size = b1_column.mean()
        average_2_values.append(int(average_size))

    # 将平均数列表转换为 NumPy 数组
    xdata = np.array(average_values)
    txdata = np.array(average_tx_size_values)
    # txdata1 = np.array(average_1_values)
    txdata2 = np.array(average_2_values)
    print(xdata)
    plt.plot(x_uniform, [int(n) /1000 for n in xdata], marker='o', color=c1,
             label=r"Baseline")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    plt.plot(x_uniform, [int(n) /1000 for n in txdata], marker='*', color=c2,
             label=r"No Optimization")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    # plt.plot(x_uniform, [int(n)  /1000for n in txdata1], marker='*', color=c3,
    #          label=r"One-Stage Optimization")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    plt.plot(x_uniform, [int(n)  /1000for n in txdata2], marker='*', color=c4,
             label=r"Optimization")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    plt.fill_between(
        x_uniform, [int(n) /1000 for n in xdata], color=c1,
        alpha=0.3  # 填充颜色的透明度
    )
    plt.fill_between(
        x_uniform, [int(n) /1000 for n in txdata], color=c2,
        alpha=0.3  # 填充颜色的透明度
    )

    plt.fill_between(
        x_uniform, [int(n) /1000 for n in txdata2], color=c4,
        alpha=0.3  # 填充颜色的透明度
    )

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigTitleSize}  # 标题的字体
    plt.xticks(x_uniform, shard_size)
    # plt.title(title, font)  # 标题
    ax = plt.gca()

    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': bigFontSize}
    ax.legend(loc="best", prop=legendFont)

    # xy坐标轴
    plt.xlabel(r'Number of Shards', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("Time(ms)", font, labelpad=labelpad)

    ax2 = ax.twinx()
    tx_base_value = xdata[0] / 1000
    print(txdata)
    print(ax.get_ylim())
    print(tx_base_value)
    ax2.set_ylim(ax.get_ylim() / tx_base_value)
    print(ax2.get_ylim())
    # ax2.plot(x_uniform, tx_ratios, color='blue', linestyle='--', label='Ratio to tx')

    # 右侧y轴标签
    ax2.set_ylabel('Ratio to BaseLine', font)

    ax.tick_params(axis='both', labelsize=bigTickSize, pad=0)
    # ax.set_xlabel('x label', fontsize=fontSize, labelpad=labelpad)
    # ax.set_ylabel('y label', fontsize=fontSize, labelpad=labelpad)
    plt.subplots_adjust(bottom=0.13)
    plt.savefig(f"t2_shard_time_{tx_size}" + ".png", dpi=1024)
    # plt.show()
    plt.close()


def shardSizeWithShardTxEvaluation(tx_size,shard_size):
    x_uniform = range(shard_size)

    tx_values = []
    average_tx_size_values = []
    to_me_values = []
    to_other_values = []

    # 根据 i 动态生成文件名
    file_name = f'{fileName}\\ShardExprimentData_{shard_size}_{tx_size}_2.xlsx'
    # 读取 Excel 文件中的指定 sheet
    df = pd.read_excel(file_name, sheet_name="shard")

    for i in range(0,shard_size) :
        b_column = df.iloc[1:, 9+i*4]
        sum = b_column.sum()
        tx_values.append(int(sum))

        b_column = df.iloc[1:, 9+i*4+2]
        sum = b_column.sum()
        to_me_values.append(int(sum))

        b_column = df.iloc[1:, 9+i*4+3]
        sum = b_column.sum()
        to_other_values.append(int(sum))

    if shard_size < 100 :
        plt.plot(x_uniform, [int(n)/1000/1000 for n in tx_values], marker='o', color=c1,
        label=r"TT-This Shard")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
    # plt.plot(x_uniform, [int(n) for n in txdata], marker='*', color=c2,
    #          label=r"no compress")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
        plt.plot(x_uniform, [int(n / 2)/1000/1000 for n in to_me_values], marker='*', color=c3,
        label=r"TR-From Other Shards")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
        plt.plot(x_uniform, [int(n)/1000/1000 for n in to_other_values], marker='p', color=c4,
        label=r"TS-To Other Shards")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接

    if shard_size > 100:
        plt.plot(x_uniform, [int(n)/1000/1000 for n in tx_values],  color=c1,
                 label=r"TT-This Shard")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
        # plt.plot(x_uniform, [int(n) for n in txdata], marker='*', color=c2,
        #          label=r"no compress")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
        plt.plot(x_uniform, [int(n / 2)/1000/1000 for n in to_me_values],  color=c3,
                 label=r"TR-From Other Shards")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接
        plt.plot(x_uniform, [int(n)/1000/1000 for n in to_other_values],  color=c4,
                 label=r"TS-To Other Shards")  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接

    # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigTitleSize}  # 标题的字体


    ax = plt.gca()
    # 图例
    legendFont = {'family': 'Times New Roman', 'weight': 'normal', 'size': bigFontSize}
    ax.legend(loc="best", prop=legendFont)

    # xy坐标轴
    plt.xlabel(r'Shard Number', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("Number of Transactions (Million)", font, labelpad=labelpad)

    ax.tick_params(axis='both', labelsize=bigTickSize, pad=0)
    plt.subplots_adjust(bottom=0.13,right=0.87)
    plt.savefig(f"t2_shard_distribution_{shard_size}" + ".png", dpi=1024)
    # plt.show()
    plt.close()


def shardSizeWithShardTxSizeBoxEvaluation(tx_size,shard_size):

    tx_size_values = []

    # 根据 i 动态生成文件名
    file_name = f'{fileName}\\ShardExprimentData_{shard_size}_{tx_size}_2.xlsx'
    # 读取 Excel 文件中的指定 sheet
    df = pd.read_excel(file_name, sheet_name="shard")

    for i in range(0,shard_size) :
        b_column = df.iloc[1:, 9+i*4+1]
        b_column = b_column / 1024  # 将列中的每个元素除以 1000
        tx_size_values.append(b_column)

    box =  plt.boxplot(tx_size_values, vert=True, patch_artist=True, boxprops=dict(facecolor='lightblue'))  # 以x为横坐标，y为纵坐标作图，直线/平滑曲线连接

    colors = [c1, c2, c3]
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
        flier.set(marker='o', color='red', alpha=0.5,markersize = 4)

        # # 字体
    font = {'family': 'Times New Roman', 'color': 'black', 'size': bigTitleSize}  # 标题的字体


    ax = plt.gca()

    # xy坐标轴
    plt.xlabel(r'Shard Number', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("Total Data Size (KB)", font, labelpad=labelpad)

    # ax.xaxis.set_visible(False)
    ax.set_xticklabels([])
    ax.set_xticks([])

    ax.tick_params(axis='both', labelsize=bigTickSize, pad=0)
    plt.subplots_adjust(bottom=0.13,left=0.15)
    plt.savefig(f"t2_shard_dataSize_{shard_size}" + ".png", dpi=1024)
    plt.show()
    plt.close()



def shardSizeWithShardTotalTxSizeBoxEvaluation(tx_size):
    plt.figure(figsize=(8, 4))  # 宽度设置为8，高度设置为4（根据需要调整）

    # 这里插入你的绘图代码
    shard_size = [4, 8, 16, 32, 64, 128, 256, 512]
    total_tx_size_values = []

    for k in shard_size:
        # 根据 i 动态生成文件名
        file_name = f'{fileName}\\ShardExprimentData_{k}_{tx_size}_2.xlsx'
        # 读取 Excel 文件中的指定 sheet
        df = pd.read_excel(file_name, sheet_name="shard")
        shard_size_values = []
        for i in range(0, k):
            b_column = df.iloc[1:, 9 + i * 4 + 1]
            b_column = b_column / 1024
            shard_size_values.append(int(b_column.mean()))
        total_tx_size_values.append(shard_size_values)

    shard_size = [0, 4, 8, 16, 32, 64, 128, 256, 512]
    x_uniform = range(len(shard_size))
    box = plt.boxplot(total_tx_size_values, vert=True, patch_artist=True, boxprops=dict(facecolor='lightblue'))

    # 设置颜色
    colors = [c1, c2, c3]
    for boxes in box['boxes']:
        boxes.set_color(c1)

    for median in box['medians']:
        median.set_color("black")

    for whisker in box['whiskers']:
        whisker.set_color("black")

    for cap in box['caps']:
        cap.set_color('gray')

    for flier in box['fliers']:
        flier.set(marker='o', color='red', alpha=0.5, markersize=4)

    # 设置标签和网格
    plt.xticks(x_uniform, shard_size)
    font = {'family': 'Times New Roman', 'color': 'black', 'size': titleSize}
    plt.xlabel(r'Number of Shards', font, labelpad=labelpad)
    plt.grid(linestyle='--', axis="y")
    plt.ylabel("Data Size (KB)", font, labelpad=labelpad)

    ax = plt.gca()
    ax.tick_params(axis='both', labelsize=tickSize, pad=0)
    plt.subplots_adjust(bottom=0.13)

    # 保存并显示图像
    plt.savefig(f"t2_shard_total_distribution_{tx_size}.png", dpi=1024)
    plt.show()
    plt.close()

if __name__ == '__main__':
    shardSizeWithNetEvaluationAppendShard(2500)
    shardSizeWithNetEvaluationAppendShard(5000)
    shardSizeWithNetEvaluationAppendShard(10000)
    shardSizeWithNetEvaluationAppendShard(20000)

    # #
    shardSizeWithTimeEvaluation(2500)
    shardSizeWithTimeEvaluation(5000)
    shardSizeWithTimeEvaluation(10000)
    shardSizeWithTimeEvaluation(20000)

    shardSizeWithShardTxEvaluation(10000,8)
    shardSizeWithShardTxEvaluation(10000,32)
    shardSizeWithShardTxEvaluation(10000,128)
    shardSizeWithShardTxEvaluation(10000,512)

    shardSizeWithShardTxSizeBoxEvaluation(10000,16)
    shardSizeWithShardTxSizeBoxEvaluation(10000,64)

    shardSizeWithShardTotalTxSizeBoxEvaluation(5000)
