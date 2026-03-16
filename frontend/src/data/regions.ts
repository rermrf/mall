export interface CascadeOption {
  label: string
  value: string
  children?: CascadeOption[]
}

export const regionData: CascadeOption[] = [
  {
    label: '北京市', value: '北京市',
    children: [{ label: '北京市', value: '北京市', children: [
      { label: '东城区', value: '东城区' },
      { label: '西城区', value: '西城区' },
      { label: '朝阳区', value: '朝阳区' },
      { label: '丰台区', value: '丰台区' },
      { label: '海淀区', value: '海淀区' },
      { label: '石景山区', value: '石景山区' },
      { label: '通州区', value: '通州区' },
      { label: '昌平区', value: '昌平区' },
      { label: '大兴区', value: '大兴区' },
      { label: '顺义区', value: '顺义区' },
    ]}],
  },
  {
    label: '上海市', value: '上海市',
    children: [{ label: '上海市', value: '上海市', children: [
      { label: '黄浦区', value: '黄浦区' },
      { label: '徐汇区', value: '徐汇区' },
      { label: '长宁区', value: '长宁区' },
      { label: '静安区', value: '静安区' },
      { label: '普陀区', value: '普陀区' },
      { label: '虹口区', value: '虹口区' },
      { label: '杨浦区', value: '杨浦区' },
      { label: '浦东新区', value: '浦东新区' },
      { label: '闵行区', value: '闵行区' },
      { label: '宝山区', value: '宝山区' },
    ]}],
  },
  {
    label: '广东省', value: '广东省',
    children: [
      { label: '广州市', value: '广州市', children: [
        { label: '天河区', value: '天河区' },
        { label: '越秀区', value: '越秀区' },
        { label: '海珠区', value: '海珠区' },
        { label: '荔湾区', value: '荔湾区' },
        { label: '番禺区', value: '番禺区' },
        { label: '白云区', value: '白云区' },
      ]},
      { label: '深圳市', value: '深圳市', children: [
        { label: '南山区', value: '南山区' },
        { label: '福田区', value: '福田区' },
        { label: '罗湖区', value: '罗湖区' },
        { label: '宝安区', value: '宝安区' },
        { label: '龙岗区', value: '龙岗区' },
        { label: '龙华区', value: '龙华区' },
      ]},
    ],
  },
  {
    label: '浙江省', value: '浙江省',
    children: [
      { label: '杭州市', value: '杭州市', children: [
        { label: '上城区', value: '上城区' },
        { label: '拱墅区', value: '拱墅区' },
        { label: '西湖区', value: '西湖区' },
        { label: '滨江区', value: '滨江区' },
        { label: '余杭区', value: '余杭区' },
        { label: '萧山区', value: '萧山区' },
      ]},
    ],
  },
  {
    label: '江苏省', value: '江苏省',
    children: [
      { label: '南京市', value: '南京市', children: [
        { label: '玄武区', value: '玄武区' },
        { label: '秦淮区', value: '秦淮区' },
        { label: '建邺区', value: '建邺区' },
        { label: '鼓楼区', value: '鼓楼区' },
        { label: '江宁区', value: '江宁区' },
      ]},
      { label: '苏州市', value: '苏州市', children: [
        { label: '姑苏区', value: '姑苏区' },
        { label: '吴中区', value: '吴中区' },
        { label: '工业园区', value: '工业园区' },
      ]},
    ],
  },
]
