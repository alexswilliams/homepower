package kl130b

//`{"smartlife.iot.smartbulb.lightingservice":{"get_light_state":{}}}`,
// {"smartlife.iot.smartbulb.lightingservice":{"get_light_state":{"on_off":1,"hue":0,"saturation":0,"color_temp":3000,"brightness":100,"mode":"normal","err_code":0}}}

// `{"smartlife.iot.smartbulb.lightingservice":{"get_preferred_state":{}}}`,
// {"smartlife.iot.smartbulb.lightingservice":{"get_preferred_state":{"states":[{"index":0,"hue":0,"saturation":0,"color_temp":2700,"brightness":50},{"index":1,"hue":0,"saturation":100,"color_temp":0,"brightness":100},{"index":2,"hue":120,"saturation":100,"color_temp":0,"brightness":100},{"index":3,"hue":240,"saturation":100,"color_temp":0,"brightness":100}],"err_code":0}}}

// `{"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{}}}`,
// {"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{"lamp_beam_angle":220,"min_voltage":220,"max_voltage":240,"wattage":10,"incandescent_equivalent":60,"max_lumens":800,"color_rendering_index":80,"err_code":0}}}

// {"smartlife.iot.smartbulb.lightingservice":{"transition_light_state":{"ignore_default":0,"mode":"normal","on_off":0,"transition_period":0}},"context":{"source":"46a4d58b-6279-432c-ae23-e115c2db8354"}}
// {"smartlife.iot.smartbulb.lightingservice":{"transition_light_state":{"brightness":50,"color_temp":2700,"hue":0,"ignore_default":1,"mode":"normal","on_off":1,"saturation":0,"transition_period":150}},"context":{"source":"46a4d58b-6279-432c-ae23-e115c2db8354"}}
// (UDP) {"smartlife.iot.smartbulb.lightingservice":{"transition_light_state":{"color_temp":6490,"hue":0,"ignore_default":1,"mode":"normal","on_off":1,"saturation":0,"transition_period":150}},"context":{"source":"46a4d58b-6279-432c-ae23-e115c2db8354"}}
// {"smartlife.iot.smartbulb.lightingservice":{"transition_light_state":{"ignore_default":1,"mode":"circadian","on_off":1,"transition_period":150}},"context":{"source":"46a4d58b-6279-432c-ae23-e115c2db8354"}}
// {"smartlife.iot.smartbulb.lightingservice":{"transition_light_state":{"brightness":100,"color_temp":0,"hue":240,"ignore_default":1,"mode":"normal","on_off":1,"saturation":100,"transition_period":150}},"context":{"source":"46a4d58b-6279-432c-ae23-e115c2db8354"}}
// (UDP) {"smartlife.iot.smartbulb.lightingservice":{"transition_light_state":{"color_temp":0,"hue":94,"ignore_default":1,"mode":"normal","on_off":1,"saturation":54,"transition_period":150}},"context":{"source":"46a4d58b-6279-432c-ae23-e115c2db8354"}}

// `{"smartlife.iot.common.emeter":{"get_daystat":{"month":7,"year":2022}}}`,
// {"smartlife.iot.common.emeter":{"get_daystat":{"day_list":[{"year":2022,"month":7,"day":15,"energy_wh":2},{"year":2022,"month":7,"day":16,"energy_wh":59},{"year":2022,"month":7,"day":17,"energy_wh":1},{"year":2022,"month":7,"day":21,"energy_wh":2},{"year":2022,"month":7,"day":23,"energy_wh":2},{"year":2022,"month":7,"day":31,"energy_wh":1},{"year":2022,"month":7,"day":1,"energy_wh":5},{"year":2022,"month":7,"day":2,"energy_wh":9},{"year":2022,"month":7,"day":3,"energy_wh":6},{"year":2022,"month":7,"day":4,"energy_wh":2},{"year":2022,"month":7,"day":5,"energy_wh":15},{"year":2022,"month":7,"day":6,"energy_wh":18},{"year":2022,"month":7,"day":8,"energy_wh":2},{"year":2022,"month":7,"day":9,"energy_wh":14},{"year":2022,"month":7,"day":10,"energy_wh":8},{"year":2022,"month":7,"day":11,"energy_wh":2},{"year":2022,"month":7,"day":14,"energy_wh":2}],"err_code":0}}}
// {"smartlife.iot.common.emeter":{"get_daystat":{"day_list":[{"year":2022,"month":8,"day":1,"energy_wh":3},{"year":2022,"month":8,"day":2,"energy_wh":1},{"year":2022,"month":8,"day":5,"energy_wh":11},{"year":2022,"month":8,"day":6,"energy_wh":9},{"year":2022,"month":8,"day":7,"energy_wh":78},{"year":2022,"month":8,"day":10,"energy_wh":3},{"year":2022,"month":8,"day":20,"energy_wh":38},{"year":2022,"month":8,"day":21,"energy_wh":12}],"err_code":0}}}
