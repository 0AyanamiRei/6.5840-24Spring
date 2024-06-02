import re


s = ["", "", "", "", "",""]

s[0]="[{0 0} {1 4952} {2 1742} {2 4467} {2 3449} {2 5845} {2 7841} {2 3693} {2 2395} {2 5352} {2 1525} {2 2441} {2 3879} {2 4497} {2 5397} {2 2787} {2 9479} {2 4216} {2 4054} {2 6926} {2 2281} {2 7939} {2 5809} {2 6196} {2 2195} {2 4189} {2 6856} {2 5134} {2 3693} {2 2548} {2 5885} {2 9558} {2 1423} {2 6112} {2 742} {2 9362} {2 9746} {2 9427} {2 209} {2 9448} {2 1965} {2 7459} {2 9702} {2 5330} {2 5434} {2 452} {2 4694} {2 1649} {2 317} {2 109} {2 1829} {2 4879} {2 437} {2 7620} {2 7097} {2 1106} {2 2589} {2 2557} {2 3775} {2 549} {2 6566} {2 6671} {2 1403} {2 2739} {2 7785} {2 8766} {2 2980} {2 7421} {2 5835} {2 2380} {2 6387} {2 5563} {2 7668} {2 1052} {2 6964} {2 8087} {2 9930} {2 1137} {2 1519} {2 8304} {2 2338} {2 7436} {2 3494} {2 2149} {2 4553} {2 7254} {2 4593} {2 65} {2 7238} {2 2959} {2 2702} {2 6254} {2 1303} {2 5124} {2 4493} {2 6434} {2 6013} {2 1945} {2 9904} {2 5436} {2 6599} {2 4225} {2 1438} {2 5761} {2 9467} {2 1475} {2 1158} {2 6466} {2 3706} {2 8328} {2 1126} {2 9227} {2 281} {2 3087} {2 3457} {2 7425} {2 2504} {2 6238} {2 7063} {2 4140} {2 59} {2 1668} {2 2411} {2 7945} {2 8130} {2 97} {2 7798} {2 8439} {2 7646} {2 8985} {2 8797} {2 6573} {2 4806} {2 3180} {2 1281} {2 7533} {2 4921} {2 4703} {2 4329} {2 222} {2 7296} {2 6811} {2 811} {2 1699} {2 2511} {2 4297} {2 7534} {2 9124} {2 8773} {2 6812} {2 3677} {2 5876} {2 4493} {2 7348} {2 997} {2 1638} {2 7327} {2 7641} {2 9717} {2 9135} {2 9588} {2 113} {2 2599} {2 4603} {2 3351} {2 5785} {2 9084} {2 8419} {2 2409} {2 2195} {2 7635} {2 6475} {2 9455} {2 9043} {2 3239} {2 8736} {2 98} {2 348} {2 656} {2 6995} {2 4446} {2 7837} {2 4080} {2 2609} {2 2004} {2 1098} {2 1517} {2 1422} {2 2174} {2 1519} {2 1427} {2 8775} {2 4630} {2 7298} {2 9075} {2 948} {2 1347} {2 56} {2 6138} {2 6687} {2 3109} {2 257} {2 2037} {2 4236} {2 4062} {2 942} {2 3020} {2 3670} {2 2730} {2 2484} {2 6659} {2 7259} {2 8450} {2 1835} {2 1796} {2 402} {2 9794} {2 6656} {2 8746} {2 7913} {2 5872} {2 5458} {2 1351} {2 1636} {2 7678} {2 8387} {2 1663} {2 7208} {2 8131} {2 9254} {2 851} {2 92} {2 3772} {2 2413} {2 6548} {2 217} {2 9605} {2 455} {2 4745} {2 5630} {2 940} {2 6436} {2 470} {2 1679} {2 8108} {2 9491} {2 3092} {2 8745} {2 4770} {2 2901} {2 5253} {2 8896} {2 7680} {2 3398} {2 2758} {2 6418} {2 3930} {2 4250} {2 9237} {2 2768} {2 9878} {2 5839} {2 3879} {2 6907} {2 5278} {2 6084} {2 645} {2 3585} {2 5670} {2 3112} {2 3806} {2 3124} {2 5112} {2 4040} {2 5230} {2 8759} {2 510} {2 3473} {2 7778} {2 6547} {2 7325} {2 5530} {2 9961} {2 5333} {2 2430} {2 3637} {2 4825} {2 9518} {2 352} {2 9428} {2 7027} {2 1039} {2 3395} {2 818} {2 5527} {2 575} {2 4725} {2 9772} {2 8370} {2 983} {2 5459} {2 1605} {2 7590} {2 8235} {2 6995} {2 3455} {2 1007} {2 2940} {2 2824} {2 5540} {2 6896} {2 3948} {2 8198} {2 1648} {2 7065} {2 8086} {2 6284} {2 9534} {2 1286} {2 2908} {2 760} {2 8823} {2 3333} {2 1628} {2 3528} {2 6721} {2 1892} {2 5102} {2 6982} {2 8817} {2 9428} {2 6459} {2 4981} {2 4811} {2 65} {2 1846} {2 2779} {2 5210} {2 1792} {2 4538} {2 4865} {2 1678} {2 7098} {2 848} {2 6435} {2 544} {2 1626} {2 8517} {2 907} {2 3016}]"
s[1]="[{0 0} {1 4952} {5 1834} {5 6605} {5 7821} {5 213}  {5 2084} {5 5076} {5 9374} {5 2467} {5 3602} {5 994} {5 5055} {5 1300} {5 1136} {5 2038} {5 3603} {5 5620} {5 4138} {5 169} {5 1410} {5 8031} {5 5137} {5 4428} {5 6055} {5 8011} {5 7576} {5 1244} {5 7511} {5 200} {5 9830} {5 4929} {5 4339} {5 1790} {5 1597} {5 2577} {5 7478} {5 6357} {5 6542} {5 5428} {5 8840} {5 3153} {5 7903} {5 8018} {5 4640} {5 4890} {5 6331} {5 9735} {5 5010} {5 6515} {5 1781} {5 7137} {5 1335} {5 9528} {5 904} {5 3259} {5 3591} {5 7967} {9 5303} {9 8856} {9 622} {9 3621} {9 3363} {9 6878} {9 754} {9 7074} {9 358} {9 4948} {9 8542} {9 1685} {9 5163} {9 2206} {9 4682} {9 114} {9 4240} {9 340} {9 3045} {9 1101} {9 2484} {9 4774} {9 3206} {9 8247} {9 7345} {17 5138} {17 1855} {17 7960} {17 4909} {17 2235} {17 3946} {17 6982} {17 5846} {17 2395} {17 5365} {17 5808} {17 8146} {17 2384} {17 2923} {17 5645} {17 7272} {17 8883} {17 7918} {17 3821} {17 6044} {17 8236} {17 8875} {17 5604} {17 7607} {17 2009} {17 2195} {17 4337} {17 9079} {17 3351} {17 3044} {17 6161} {17 7674} {17 3024} {17 5407} {17 140} {17 9114} {17 951} {17 9905} {17 9023} {17 968} {17 1486} {17 216} {17 9529} {17 2405} {17 5024} {17 622} {17 607} {17 4002} {17 484} {17 1774} {17 3343} {17 85} {17 6250} {17 8958} {17 2289} {17 7721} {17 4551} {17 7175} {17 9098} {17 1227} {17 4460} {17 6468} {17 5813} {17 6764} {17 3989} {17 6339} {17 5579} {17 9874} {17 6704} {17 2482} {17 5444} {17 1041} {17 6434} {17 34} {17 2366} {17 1754} {17 1878} {17 4318} {17 2652} {17 4532} {17 3646} {17 5708} {17 9730} {17 3516} {17 952} {17 9208} {17 2621} {17 5662} {17 5583} {17 7206} {17 6619} {17 8783} {17 2421} {17 2521} {17 519} {17 2537} {17 2129} {17 6587} {17 2550} {17 6264} {17 2637} {17 2274} {17 9448} {17 105} {17 2226} {17 7980} {17 4003} {17 3753} {17 7784} {17 4722} {17 9908} {17 8860} {17 2639} {17 1131} {17 4856} {17 1392} {17 8357} {17 9954} {17 6053} {17 7569} {17 2843} {17 1624} {17 1188} {17 1693} {17 2440} {17 521} {17 3768} {17 8355} {17 3167} {17 978} {17 6478} {17 5927} {17 3137} {17 6337} {17 4245} {17 5930} {17 6985} {17 3222} {17 2760} {17 7423} {17 6588} {17 8899} {17 6201} {17 534} {17 6348} {17 40} {17 5044} {17 7551} {17 7087} {17 4972} {17 85} {17 7592} {17 7723} {17 1993} {17 2144} {17 4367} {17 7338} {17 7585} {17 3756} {17 9323} {17 3637} {17 1434} {17 1194} {17 1641} {17 1003} {17 7385} {17 1130} {17 4212} {17 9815} {17 7823} {17 7897} {17 6010} {17 1256} {17 9046} {17 531} {17 4696} {17 4018} {17 666} {17 3256} {17 8920} {17 5660} {17 2800} {17 6896} {17 7030} {17 2389} {17 9120} {17 9524} {17 2626} {17 2754} {17 842} {17 6893} {17 4812} {17 3514} {17 6880} {17 7041} {17 7663} {17 266} {17 3582} {17 9174} {17 9397} {17 492} {17 4406} {17 4514} {17 2093} {17 3901} {17 3620} {17 6634} {17 6382} {17 5865} {17 5273} {17 6042} {17 7059} {17 8648} {17 3544} {17 6557} {17 3473} {17 760} {17 3791} {17 1189} {17 6985} {17 1179} {17 5823} {17 8176} {17 9708} {17 9160} {17 7418} {17 8629} {17 9847} {17 9008} {17 9109} {17 3293} {17 6916} {17 5375} {17 9912} {17 9076} {17 1667} {17 1382} {17 4180} {34 4038} {34 1146} {34 5102} {34 4365} {34 202} {34 5112} {34 4906} {34 9940} {34 4334} {34 6963} {34 9758} {34 8713} {34 4982} {34 8295} {34 6840} {34 1678} {34 9228} {34 9423} {34 6410} {34 9865} {34 5734} {34 9022} {34 4117} {34 8065} {34 6934} {34 5933} {34 2648} {34 5061} {34 1888} {34 7006} {34 9999} {34 7290} {34 4772} {34 9182} {34 4544} {34 8962} {34 4276} {34 6375} {34 6084} {34 1300} {34 9488} {34 5153} {34 5054} {34 8178} {34 7310} {34 1683} {34 9239} {34 3533} {34 1844} {34 4181} {34 350} {34 2462} {34 7272} {34 1116} {34 93} {34 6573} {34 8809} {34 1330} {34 1900} {34 9004} {34 380} {34 5419} {34 7442} {34 2178} {34 4230} {34 9586} {34 8998} {34 2514} {34 3377} {34 4040} {34 278} {34 4549} {34 8796} {34 1934} {34 7932} {34 1521} {34 1198} {34 3750} {34 5595} {34 3388} {34 3072} {34 8952} {34 5743} {34 915} {34 337} {34 9532} {34 5537} {34 726} {34 9288} {34 1392} {34 9330} {34 2881} {34 9392} {34 8189} {34 8272} {34 3597} {34 364} {34 9867} {34 1271} {34 110} {34 8105} {34 1295} {34 50} {34 3253} {34 609} {34 6552} {34 2930} {34 7061} {34 1899} {34 10} {34 9174} {34 191} {34 1704} {34 9841} {34 8038} {34 9612} {34 3710} {34 3554} {34 1028} {34 4509} {34 2006} {34 3257} {34 8026} {34 3246} {34 1757} {34 7481} {34 2602} {34 7986} {34 6436} {34 6189} {34 772} {34 3249} {34 1924} {34 1690} {34 7081} {34 6897} {34 5746} {34 3866} {34 7120} {34 7357} {34 915} {34 860} {34 8664} {34 938} {34 5122} {34 8060} {34 5913} {34 9075} {34 2738} {34 6276} {34 2030} {34 8210} {34 6424} {34 9658} {34 7381} {34 8071} {34 50} {34 4009} {34 225} {34 7931} {34 6492} {34 1511} {34 4884} {34 3110} {34 2243} {34 1897} {34 724} {34 8277} {34 3570} {34 8407} {34 4742} {34 9779} {34 6023} {34 3837} {34 4532} {34 2324} {34 5585} {34 5866} {34 2660} {34 6180} {34 7861} {34 812} {34 1301} {34 8304} {34 3740} {34 5029} {34 7557} {34 3471} {34 4520} {34 2640} {34 4132} {34 4532} {34 1722} {34 762} {34 1155} {34 2408} {34 7924} {34 3241} {34 6502} {34 8782} {34 5504} {34 8847} {34 5158} {34 4265} {34 8578} {34 2152} {34 6460} {34 7802} {34 831} {34 3966} {34 1023} {34 9744} {34 8669} {34 7464} {34 2950} {34 3005} {34 362} {34 9485} {34 6490} {34 1323} {34 104} {34 4313} {34 2022} {34 2319} {34 7174} {34 5240} {51 3496} {51 3106} {51 9542} {51 1411} {51 8065} {51 4624} {51 7130} {51 7899} {51 608} {51 5565} {51 9249} {51 4255} {51 3927} {51 640} {51 1478} {51 6423} {51 3007} {51 8353} {51 9659} {51 770} {51 1096} {51 5652} {51 4222} {51 6481} {51 8126} {51 4909} {51 6370} {51 306} {51 4264} {51 4474} {51 2356} {51 5884} {51 3179} {51 4146} {51 8687} {51 3199} {51 7999} {51 2614} {51 5590} {51 2050} {51 8473} {51 8299} {51 3584} {51 7264} {51 2785} {51 7438} {51 2953} {51 7723} {51 3953} {51 8018} {51 7614} {51 6746} {51 4196} {51 6909} {51 2656} {51 1475} {51 9212} {51 3184} {51 9769} {51 4662} {51 422} {51 6331} {51 6521} {51 8329} {51 5204} {51 6089} {51 4547} {51 5387} {51 2482} {51 1876} {51 8497} {51 3124} {51 1242} {51 702} {51 7258} {51 4360} {51 8950} {51 2177} {51 5828} {51 9260} {51 9277} {51 4418} {51 4618} {51 728} {51 966} {51 1447} {51 912} {51 6038} {51 8513}]"
s[2]="[{0 0} {1 4952} {5 1834} {5 6605} {5 7821} {5 213}  {5 2084} {5 5076} {5 9374} {5 2467} {5 3602} {5 994} {5 5055} {5 1300} {5 1136} {5 2038} {5 3603} {5 5620} {5 4138} {5 169} {5 1410} {5 8031} {5 5137} {5 4428} {5 6055} {5 8011} {5 7576} {5 1244} {5 7511} {5 200} {5 9830} {5 4929} {5 4339} {5 1790} {5 1597} {5 2577} {5 7478} {5 6357} {5 6542} {5 5428} {5 8840} {5 3153} {5 7903} {5 8018} {5 4640} {5 4890} {5 6331} {8 355} {8 1583} {8 8557} {8 435} {8 8811} {8 5336} {8 2983} {8 9387} {8 9800} {8 4553} {8 1688} {8 4340} {8 4463} {8 4718} {8 3830} {8 1768} {8 5785} {8 9478} {8 1446} {8 5207} {8 1094} {8 359} {8 84} {8 4669} {8 3996} {8 451} {8 823} {8 5099} {8 6691} {8 5679} {8 6548} {8 4017} {8 2669} {8 1440} {8 6545} {8 9234} {8 4092} {8 907} {8 9104} {8 765} {8 3004} {8 6052} {8 3659} {8 8040} {8 9177} {8 101} {8 1361} {8 2809} {8 717} {8 1238} {8 981} {8 3262} {8 5278} {8 1901} {8 5611} {8 1985} {8 3603} {8 8208} {8 8038} {8 6052} {8 6937} {8 4285} {8 8907} {8 3339} {8 5780} {8 3576} {8 617} {8 1502} {8 6807} {8 9772} {8 4602}]"
s[3]="[{0 0} {1 4952} {5 1834} {5 6605} {5 7821} {5 213}  {5 2084} {5 5076} {5 9374} {5 2467} {5 3602} {5 994} {5 5055} {5 1300} {5 1136} {5 2038} {5 3603} {5 5620} {5 4138} {5 169} {5 1410} {5 8031} {5 5137} {5 4428} {5 6055} {5 8011} {5 7576} {5 1244} {5 7511} {5 200} {5 9830} {5 4929} {5 4339} {5 1790} {5 1597} {5 2577} {5 7478} {5 6357} {5 6542} {5 5428} {5 8840} {5 3153} {5 7903} {5 8018} {5 4640} {5 4890} {5 6331} {5 9735} {5 5010} {5 6515} {5 1781} {5 7137} {5 1335} {5 9528} {5 904} {5 3259} {5 3591} {5 7967} {9 5303} {9 8856} {9 622} {9 3621} {9 3363} {9 6878} {9 754} {9 7074} {9 358} {9 4948} {9 8542} {9 1685} {9 5163} {9 2206} {9 4682} {9 114} {9 4240} {9 340} {9 3045} {9 1101} {9 2484} {9 4774} {9 3206} {9 8247} {9 7345} {17 5138} {20 4263} {20 515} {20 1865} {20 2143} {20 827} {20 6813} {20 5604} {20 2611} {20 2868} {20 7131} {20 3595} {20 1641} {20 2651} {20 832} {20 7740} {20 8020} {20 5423} {20 6034} {20 5462} {20 5019} {20 8217} {20 1299} {20 1123} {20 1353} {20 8856} {20 397} {20 8707} {20 8495} {20 2988} {20 8273} {20 2334} {20 6672} {20 8451} {20 5878} {20 9637} {20 908} {20 8026} {20 4615} {20 3832} {20 3648} {20 7836} {20 8607} {20 3505} {20 2152} {20 8831} {20 2853} {20 5931} {20 685} {20 6467} {20 7198} {20 8810} {20 5016} {20 1014} {20 7764} {20 6931} {20 2932} {20 5719} {20 7920} {20 9642} {20 8323} {20 9348} {20 2463} {20 6451} {20 2593} {20 6150} {20 3183} {20 9057} {20 2696} {20 4232} {20 7838} {20 6169} {20 4835} {20 2487} {20 9277} {20 5917} {20 2034} {20 5205} {20 2559} {20 9561} {20 6985} {20 375} {20 7265} {20 6594} {20 584} {20 7582} {20 4273} {20 3032} {20 5780} {28 3241} {28 4199} {28 9727} {28 1374} {28 1250} {28 5464} {28 5994} {28 7224} {28 8579} {28 2602} {28 3267} {28 2508} {28 8642} {28 3613} {28 7533} {28 1808} {28 9953} {28 6038} {28 4781} {28 5805} {28 6808} {28 2090} {28 4538} {28 2467} {28 2960} {28 3826} {28 2068} {28 2942} {28 7104} {28 8430} {28 9050} {28 8202} {28 4249} {28 2833} {28 4093} {28 3570} {28 1917} {28 8886} {28 9958} {28 3612} {28 8901} {28 3479} {28 5724} {28 7891} {28 8086} {28 2889} {28 5422} {28 2773} {28 2051} {28 5426} {28 9090} {28 954} {28 1485} {28 9984} {28 4643} {28 9385} {28 4601} {28 276} {28 1915} {28 9876} {28 742} {28 7975} {28 765} {28 5201} {28 3255} {28 4582} {28 5582} {28 8446} {28 6161} {28 3525} {28 7636} {28 5163} {28 1745} {28 8283} {28 3846} {28 6730} {28 9835} {28 7381} {28 1904} {28 9936} {28 778} {28 1428} {28 380} {28 1813} {28 2831} {28 6200} {28 1825} {28 1357} {28 7772} {28 8278} {28 72} {28 2579} {28 5784} {28 2114} {28 709} {28 566} {28 5782} {28 6105} {28 5170} {28 5192} {28 4724} {28 3682} {28 8547} {28 9306} {28 1270} {28 5646} {28 9900} {28 6526} {28 1594} {28 347} {28 3873} {28 9005} {28 9459} {28 2869} {28 154} {28 6938} {28 8753} {28 9965} {28 2008} {28 6510} {28 9922} {28 4948} {28 2841} {28 27} {28 3712} {28 1079} {28 2133} {28 3736} {28 665} {28 4375} {28 6325} {28 5734} {28 9280} {28 4956} {28 5687} {28 572} {28 2744} {28 740} {28 871} {28 6822} {28 3883} {28 5965} {28 3477} {28 2494} {28 8267} {28 8351} {28 1426} {28 9265} {28 1146} {28 6182} {28 6170} {28 885} {28 888} {28 5762} {28 5675} {28 8680} {28 655} {28 685} {28 1837} {28 8440} {28 545} {28 2140} {28 7200} {28 3152} {28 1345} {28 4627} {28 2285} {28 140} {28 4180} {28 3950} {28 6292} {28 2675} {28 7757} {28 1490} {28 4960} {28 6147} {28 9661} {28 3136} {28 793} {28 4177} {28 2520} {28 3685} {28 646} {28 2680} {28 1851} {28 2230} {28 6207} {28 1625} {28 9436} {28 8213} {28 5373} {28 2424} {28 6891} {28 6815} {28 2485} {28 4122} {28 9813} {28 1696} {28 2361} {28 450} {28 9308} {28 2576} {28 3081} {28 129} {28 9518} {28 8995} {28 594} {28 9132} {28 6132} {28 5072} {28 5467} {28 1388} {28 4598} {28 2371} {28 7662} {28 3061} {28 9606} {28 5476} {28 3051} {28 4237} {28 58} {28 233} {28 3131} {28 797} {28 5804} {28 2995} {28 5985} {28 6916} {28 5015} {28 7184} {28 7603} {28 9276} {28 4708} {28 4364} {28 3823} {28 8808} {28 2131} {28 7920} {28 9913} {28 1458} {28 3360} {28 9505} {28 5804} {28 2392} {28 4282} {28 918} {28 4648} {28 2262} {28 2444} {28 3271} {28 5654} {28 9005} {28 1606} {28 9241} {28 2473} {28 5217} {28 5202} {28 4148} {28 713} {28 770} {28 5615} {28 5393} {28 9895} {28 3394} {28 8420} {28 9219} {28 4085} {28 3857} {28 8746} {28 7846} {28 1512} {28 8311} {28 6426} {28 6077} {28 519} {28 8264} {28 3067} {28 3891} {28 413} {28 6529} {28 1659} {28 3831} {28 4588} {28 2733} {28 2793} {28 8394} {28 2704} {28 1029} {28 265} {28 9706} {28 902} {28 1654} {28 9288} {28 4715} {28 4368} {28 998} {28 6002} {28 661} {28 7986} {28 7676} {28 7572} {28 9725} {28 4967} {28 5045} {28 2257} {28 1650} {28 4376} {28 3981} {28 4058} {28 1746} {28 3914} {28 8115} {28 3661} {28 49} {28 5557} {28 6783} {28 3232} {28 2947} {28 1379} {28 3859} {28 5695} {28 6132} {28 5217} {28 524} {28 1318} {28 1286} {28 9572} {28 7735} {28 579} {28 2891} {28 7212} {28 5346} {28 6883} {28 4513} {28 8208} {28 6791} {28 1205} {28 7007} {28 1066} {28 4934} {28 1697} {28 5224} {28 6844} {28 8786} {28 889} {28 1659} {28 2764} {28 8677} {28 9848} {28 449} {28 1301} {28 5966} {28 3503} {28 2047} {28 5251} {28 1756} {28 6380} {28 6066} {28 4814} {28 1167} {28 8277} {28 101} {28 19} {28 8289} {28 595} {28 51} {28 2458} {28 5844} {28 6852} {28 410} {28 5791} {28 1584} {28 8758} {28 2976} {28 1269} {28 8820} {28 4724} {28 1049} {28 3051} {28 5515} {28 1302} {28 848} {28 9865} {28 7447} {28 4828} {28 7827} {28 1390} {28 7298} {28 2514} {28 5475} {28 4938} {28 3965} {28 2194} {28 877} {28 3823} {28 8777} {28 451} {28 5776} {28 7611} {28 7413} {28 4856} {28 2354} {28 4959} {28 2422} {28 8206} {28 4921} {28 471} {28 9714} {28 2056} {28 2729} {28 980} {28 1538} {28 4114} {28 1211} {28 505} {28 5342} {28 6728} {28 1050} {28 674} {28 5548} {28 4559} {28 9214} {28 1644} {28 6898} {28 8725} {28 5174} {28 5771} {28 3758} {28 2530} {28 1792} {28 5503} {28 3780} {28 5892} {28 1516} {28 5810} {28 1922} {28 7344} {28 7645} {28 1450} {28 4268} {28 5051} {28 5712} {28 9643} {28 4603} {28 9424} {28 2489} {28 6270} {28 4151} {28 6162} {28 8227} {28 6684} {28 2146} {28 6821} {28 1877} {28 8771} {28 1106} {28 3705} {28 468} {28 6553} {28 4222} {28 63} {28 4555} {28 7678} {28 3333} {28 7164} {28 4021} {28 5939} {28 8612} {28 3432} {28 6732} {28 1131} {28 4830} {28 6449} {28 8152} {28 5034} {28 133} {28 642} {28 7637} {28 4059} {28 603} {28 6289} {28 2251} {28 1750} {28 6181} {28 2514} {28 641} {28 7033} {28 3856} {28 3913} {28 3308} {28 9499} {28 8147} {28 8697} {28 6773} {28 3006} {28 9161} {28 9355} {28 8406} {28 7412} {28 8840} {28 4228} {28 6632} {28 3944} {28 1257} {28 6911} {28 7356} {28 7711} {28 9630} {28 4357}]"
s[4]="[{0 0} {1 4952} {5 1834} {5 6605} {5 7821} {5 213}  {5 2084} {5 5076} {5 9374} {5 2467} {5 3602} {5 994} {5 5055} {5 1300} {5 1136} {5 2038} {5 3603} {5 5620} {5 4138} {5 169} {5 1410} {5 8031} {5 5137} {5 4428} {5 6055} {5 8011} {5 7576} {5 1244} {5 7511} {5 200} {5 9830} {5 4929} {5 4339} {5 1790} {5 1597} {5 2577} {5 7478} {5 6357} {5 6542} {5 5428} {5 8840} {5 3153} {5 7903} {5 8018} {5 4640} {5 4890} {5 6331} {5 9735} {5 5010} {5 6515} {5 1781} {5 7137} {5 1335} {5 9528} {5 904} {5 3259} {5 3591} {5 7967} {9 5303} {9 8856} {9 622} {9 3621} {9 3363} {9 6878} {9 754} {9 7074} {9 358} {9 4948} {9 8542} {9 1685} {9 5163} {9 2206} {9 4682} {9 114} {9 4240} {9 340} {9 3045} {9 1101} {9 2484} {9 4774} {9 3206} {9 8247} {9 7345} {17 5138} {20 4263} {20 515} {20 1865} {20 2143} {20 827} {20 6813} {20 5604} {20 2611} {20 2868} {20 7131} {20 3595} {20 1641} {20 2651} {20 832} {20 7740} {20 8020} {24 201} {24 7517} {24 9834} {24 9375} {24 877} {24 5783} {24 5031} {24 618} {24 645} {24 3916} {24 5195} {24 4461} {24 2514} {24 8392} {24 3968} {24 7657} {24 6022} {24 65} {24 5806} {24 4309} {24 1061} {24 7470} {24 422} {24 231} {24 2261} {24 3110} {24 2844} {24 6470} {24 3379} {24 3399} {24 4951} {24 5569} {24 954} {24 914} {24 6795} {24 2252} {24 7915} {24 4554} {24 2998} {24 9943} {24 2266} {24 8212} {24 8393} {24 326} {24 286} {24 4596} {24 2566} {24 4194} {24 2006} {24 1645} {24 7662} {24 8822} {24 9930} {24 2530} {24 6167} {24 1806} {24 8666} {24 8397} {24 7950} {24 6226} {24 8390} {24 3911} {24 1307} {24 5297} {24 8668} {24 6657} {24 7911} {24 5284} {24 3087} {24 640} {24 2823} {24 5261} {24 9452} {24 6196} {24 2839} {24 420} {24 8640} {24 5766} {24 703} {24 1333} {24 7491} {24 7086} {24 8389} {24 136} {24 2271} {24 5186} {47 5348} {47 5982} {47 7347} {47 3730} {47 4654} {47 8489} {47 4649} {47 4236} {47 7046} {47 8908} {47 574} {47 1882} {47 7274} {47 3606} {47 4345} {47 9764} {47 3674} {47 6144} {47 7428} {47 8842} {47 1765} {47 24} {47 7508} {47 3301} {47 4186} {47 4194} {47 7877} {47 9275} {47 3860} {47 8857} {47 5406} {47 3534} {47 3019} {47 7349} {47 4786} {47 6357} {47 5112} {47 7147} {47 6125} {47 3534} {47 5815} {47 5050} {47 1896} {47 9323} {47 5094} {47 7269} {47 8947} {47 2351} {47 4569} {47 8501} {47 2168} {47 3005} {47 3819} {47 1052} {47 8507} {47 6233} {47 277} {47 4712} {47 645} {47 5261} {47 2980} {47 4879} {47 6299} {47 5886} {47 211} {47 6453} {47 1054} {47 6904} {47 4750} {47 596} {47 9227} {47 6682} {47 7844} {47 7391} {47 8666} {47 8544} {47 5946} {47 2677} {47 9368} {47 3973} {47 5749} {47 5912} {47 6414} {47 9781} {47 8263} {47 7766} {47 2618} {47 7983} {47 2617} {47 652} {47 103} {47 6207} {47 4442} {47 6316} {47 5632} {47 7565} {47 4748} {47 213} {47 1920} {47 6916} {47 6754} {47 7969} {47 9837} {47 7940} {47 8989} {47 759} {47 1825} {47 6793} {47 3608} {47 4309} {47 3649} {47 2110} {47 1086} {47 1766} {47 1258} {47 9642} {47 9118} {47 8886} {47 7604} {47 6517} {47 3278} {47 7785} {47 9502} {47 1402} {47 8488} {47 1711} {47 2555} {47 8972} {47 4296} {47 6583} {47 3609} {47 6915} {47 9523} {47 5127} {47 2788} {47 285} {47 377} {47 9478} {47 2626} {47 4134} {47 1549} {47 6028} {47 3102} {47 9990} {47 5304} {47 3743} {47 8408} {47 6551} {47 4142} {47 6859} {47 5776} {47 8706} {53 5205} {53 4578} {53 2444} {53 5015} {53 6165} {53 2472} {53 2280} {53 6790} {53 3771} {53 9153} {53 9675} {53 3594} {53 8120} {53 3375} {53 6868} {53 6154} {53 2370} {53 358} {53 3650} {53 4667} {53 3364} {53 3142} {53 7917} {53 8916} {53 5931} {53 1711} {53 4713} {53 5509} {53 3346} {53 2585}]"

import re

log_dict ={}
logs = [[] for _ in range(5)]

def extract_dict(s, i):
    matches = re.findall(r'\{(\d+) (\d+)\}', s)
    for a, b in matches:
        key = f"{a} {b}"
        logs[i].append(key)
        if key in log_dict:
            log_dict[key] += 1
        else:
            log_dict[key] = 1 

for i, v in enumerate(s):
    extract_dict(v, i)

for i in range(5):
    cnt = 0
    for v in logs[i]:
        if log_dict[v] >= 3:
            cnt+=1
        else:
            break
    print("s", i, " ",cnt)