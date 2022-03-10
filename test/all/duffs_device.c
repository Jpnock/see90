int f(int x, int y)
{
    int i_to = 10;
    int *to = &i_to;
    int count = 10;
    int n = (count + 7) / 8;
    int b = 1;

    switch (count % 8)
    {
    case 0:
        do
        {
            *to += 1;
        case 7:
            *to += 1;
        case 6:
            *to += 1;
        case 5:
            *to += 1;
        case 4:
            *to += 1;
        case 3:
            *to += 1;
        case 2:
            *to += 1;
        case 1:
            *to += 1;
            switch (n)
            {
            case 2:
                *to += 10;
                break;
            }
        default:;
        } while (--n > 0);
    }
    return i_to;
}
