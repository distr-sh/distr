export type Metric = {
  value: string;
  label: string;
  icon: string;
};

export const metrics: Metric[] = [
  {value: '200+', label: 'Organizations', icon: 'lucide:building-2'},
  {value: '5000+', label: 'Agent Downloads', icon: 'lucide:download'},
  {value: '1100+', label: 'GitHub Stars', icon: 'lucide:star'},
  {value: '20+', label: 'Contributors', icon: 'lucide:git-pull-request'},
];
